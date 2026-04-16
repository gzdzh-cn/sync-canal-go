// =================================================================================
// Monitor Collector - 指标采集器
// =================================================================================

package monitor

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/gogf/gf/v2/frame/g"

	"sync-canal-go/internal/model/entity"
	"sync-canal-go/internal/service"
)

// Collector 指标采集器
type Collector struct {
	config     *entity.MonitorConfig
	store      service.IStore
	running    atomic.Bool
	stopChan   chan struct{}
	mu         sync.RWMutex

	// 实时指标
	status     *entity.ServiceStatus
	targets    map[string]*entity.TargetStatus
	eventCount atomic.Int64
	errorCount atomic.Int64
	qpsCounter atomic.Int64
	tpsCounter atomic.Int64

	// 环形缓冲区
	eventBuffer *RingBuffer[*entity.SyncEvent]
	errorBuffer *RingBuffer[*entity.SyncError]
}

// NewCollector 创建采集器
func NewCollector(config *entity.MonitorConfig, store service.IStore) *Collector {
	config.SetDefaults()
	return &Collector{
		config:       config,
		store:        store,
		stopChan:     make(chan struct{}),
		status:       &entity.ServiceStatus{Status: "starting", StartTime: time.Now()},
		targets:      make(map[string]*entity.TargetStatus),
		eventBuffer:  NewRingBuffer[*entity.SyncEvent](config.MaxEvents),
		errorBuffer:  NewRingBuffer[*entity.SyncError](config.MaxErrors),
	}
}

// Init 初始化/更新配置
func (c *Collector) Init(config *entity.MonitorConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = config

	// 动态更新缓冲区大小
	if config.MaxEvents > 0 && c.eventBuffer == nil {
		c.eventBuffer = NewRingBuffer[*entity.SyncEvent](config.MaxEvents)
	} else if config.MaxEvents > 0 && c.eventBuffer.Capacity() != config.MaxEvents {
		// 创建新的缓冲区（旧数据会丢失，但这是预期行为）
		c.eventBuffer = NewRingBuffer[*entity.SyncEvent](config.MaxEvents)
	}

	if config.MaxErrors > 0 && c.errorBuffer == nil {
		c.errorBuffer = NewRingBuffer[*entity.SyncError](config.MaxErrors)
	} else if config.MaxErrors > 0 && c.errorBuffer.Capacity() != config.MaxErrors {
		c.errorBuffer = NewRingBuffer[*entity.SyncError](config.MaxErrors)
	}

	return nil
}

// Start 启动采集器
func (c *Collector) Start() error {
	if c.running.Load() {
		return nil
	}
	c.running.Store(true)
	c.status.Status = "running"
	c.status.StartTime = time.Now()

	// 启动定时采集协程
	go c.collectLoop()

	// 启动指标计算协程
	go c.metricsLoop()

	g.Log().Infof(context.Background(), "Monitor collector started")
	return nil
}

// Stop 停止采集器
func (c *Collector) Stop() error {
	if !c.running.Load() {
		return nil
	}
	c.running.Store(false)
	close(c.stopChan)
	c.status.Status = "stopped"
	g.Log().Infof(context.Background(), "Monitor collector stopped")
	return nil
}

// collectLoop 定时采集循环
func (c *Collector) collectLoop() {
	ticker := time.NewTicker(time.Duration(c.getCollectPeriod()) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			// 如果监控已禁用，跳过采集
			if !c.isEnabled() {
				continue
			}
			c.collectMetrics()
			// 动态调整采集周期
			ticker.Reset(time.Duration(c.getCollectPeriod()) * time.Second)
		}
	}
}

// getCollectPeriod 获取采集周期
func (c *Collector) getCollectPeriod() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.config == nil || c.config.CollectPeriod <= 0 {
		return 10 // 默认 10 秒
	}
	return c.config.CollectPeriod
}

// metricsLoop 指标计算循环
func (c *Collector) metricsLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastEventCount, lastTpsCount int64

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			// 计算QPS和TPS
			currentEventCount := c.eventCount.Load()
			currentTpsCount := c.tpsCounter.Load()
			c.status.QPS = float64(currentEventCount - lastEventCount)
			c.status.TPS = float64(currentTpsCount - lastTpsCount)
			lastEventCount = currentEventCount
			lastTpsCount = currentTpsCount

			// 更新运行时长
			c.status.Uptime = int64(time.Since(c.status.StartTime).Seconds())
		}
	}
}

// collectMetrics 采集指标
func (c *Collector) collectMetrics() {
	ctx := context.Background()

	// 采集位置信息
	// TODO: 从 canal 获取当前位置

	// 批量写入历史数据
	events := c.eventBuffer.GetAll()
	if len(events) > 0 && c.store != nil {
		if err := c.store.SaveEvents(ctx, events); err != nil {
			g.Log().Warningf(ctx, "Save events failed: %v", err)
		}
	}
}

// isEnabled 检查监控是否启用
func (c *Collector) isEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config != nil && c.config.Enabled
}

// OnEvent 处理事件
func (c *Collector) OnEvent(e *entity.SyncEvent) {
	// 如果监控已禁用，跳过处理
	if !c.isEnabled() {
		return
	}

	c.eventCount.Add(1)
	// 兼容大小写：canal 库的 Action 类型是小写 (insert/update/delete)
	upperType := strings.ToUpper(e.EventType)
	if upperType == "INSERT" || upperType == "UPDATE" || upperType == "DELETE" {
		c.tpsCounter.Add(1)
	}

	// 写入环形缓冲区
	c.eventBuffer.Put(e)

	// 写入存储
	if c.store != nil {
		go func() {
			ctx := context.Background()
			if err := c.store.SaveEvent(ctx, e); err != nil {
				g.Log().Warningf(ctx, "Save event failed: %v", err)
			}
		}()
	}
}

// OnPosition 处理位置更新
func (c *Collector) OnPosition(p *entity.SyncPosition) {
	// 如果监控已禁用，跳过处理
	if !c.isEnabled() {
		return
	}

	c.mu.Lock()
	c.status.BinlogFile = p.BinlogFile
	c.status.BinlogPos = p.BinlogPos
	c.status.GTID = p.GTID
	c.status.DelaySeconds = p.DelaySeconds
	c.mu.Unlock()

	// 写入存储
	if c.store != nil {
		go func() {
			ctx := context.Background()
			if err := c.store.SavePosition(ctx, p); err != nil {
				g.Log().Warningf(ctx, "Save position failed: %v", err)
			}
		}()
	}
}

// OnError 处理错误
func (c *Collector) OnError(err *entity.SyncError) {
	// 如果监控已禁用，跳过处理
	if !c.isEnabled() {
		return
	}

	c.errorCount.Add(1)

	// 写入环形缓冲区
	c.errorBuffer.Put(err)

	// 更新目标状态
	c.mu.Lock()
	if target, ok := c.targets[err.TargetName]; ok {
		target.TotalErrors++
		target.LastError = err.Message
	}
	c.mu.Unlock()

	// 写入存储
	if c.store != nil {
		go func() {
			ctx := context.Background()
			if e := c.store.SaveError(ctx, err); e != nil {
				g.Log().Warningf(ctx, "Save error failed: %v", e)
			}
		}()
	}
}

// OnMetric 处理指标
func (c *Collector) OnMetric(m *entity.SyncMetric) {
	// 如果监控已禁用，跳过处理
	if !c.isEnabled() {
		return
	}

	if c.store != nil {
		go func() {
			ctx := context.Background()
			if err := c.store.SaveMetric(ctx, m); err != nil {
				g.Log().Warningf(ctx, "Save metric failed: %v", err)
			}
		}()
	}
}

// OnRow 处理 canal 行事件
func (c *Collector) OnRow(e *canal.RowsEvent, durationMs int, err error) {
	event := &entity.SyncEvent{
		Timestamp:  time.Now(),
		EventType:  string(e.Action),
		TargetName: e.Table.Schema,
		TableName:  e.Table.Name,
		RowsCount:  len(e.Rows),
		BinlogFile: "", // TODO: 从 canal 获取
		BinlogPos:  0,  // TODO: 从 canal 获取
		DurationMs: durationMs,
		Success:    err == nil,
	}
	if err != nil {
		event.ErrorMsg = err.Error()
	}
	c.OnEvent(event)
}

// RegisterTarget 注册目标
func (c *Collector) RegisterTarget(name, targetType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.targets[name] = &entity.TargetStatus{
		Name:      name,
		Type:      targetType,
		Status:    "connected",
		IsEnabled: true,
	}
}

// UpdateTargetStatus 更新目标状态
func (c *Collector) UpdateTargetStatus(name, status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if target, ok := c.targets[name]; ok {
		target.Status = status
		target.LastSyncTime = time.Now()
	}
}

// GetStatus 获取状态
func (c *Collector) GetStatus() *entity.ServiceStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	status := *c.status
	return &status
}

// GetTargets 获取所有目标状态
func (c *Collector) GetTargets() []*entity.TargetStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]*entity.TargetStatus, 0, len(c.targets))
	for _, t := range c.targets {
		tCopy := *t
		result = append(result, &tCopy)
	}
	return result
}

// GetEventBuffer 获取事件缓冲区
func (c *Collector) GetEventBuffer() []*entity.SyncEvent {
	return c.eventBuffer.GetAll()
}

// GetErrorBuffer 获取错误缓冲区
func (c *Collector) GetErrorBuffer() []*entity.SyncError {
	return c.errorBuffer.GetAll()
}

// GetConfig 获取当前配置
func (c *Collector) GetConfig() *entity.MonitorConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.config == nil {
		return &entity.MonitorConfig{}
	}
	// 返回副本
	config := *c.config
	return &config
}

// SetEnabled 设置监控启用状态
func (c *Collector) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.config != nil {
		c.config.Enabled = enabled
	}
}
