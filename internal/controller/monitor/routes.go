// =================================================================================
// Monitor Routes - 路由注册
// =================================================================================

package monitor

import (
	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"sync-canal-go/internal/logic/monitor"
	"sync-canal-go/internal/model/entity"
	"sync-canal-go/internal/service"
)

// CollectorWrapper 采集器包装器，实现 sync.ICollector 接口
type CollectorWrapper struct {
	Collector service.ICollector
}

// OnEvent 实现 ICollector 接口
func (w *CollectorWrapper) OnEvent(e *entity.SyncEvent) {
	if w.Collector != nil {
		w.Collector.OnEvent(e)
	}
}

// OnPosition 实现 ICollector 接口
func (w *CollectorWrapper) OnPosition(p *entity.SyncPosition) {
	if w.Collector != nil {
		w.Collector.OnPosition(p)
	}
}

// OnError 实现 ICollector 接口
func (w *CollectorWrapper) OnError(err *entity.SyncError) {
	if w.Collector != nil {
		w.Collector.OnError(err)
	}
}

// RegisterTarget 实现 ICollector 接口
func (w *CollectorWrapper) RegisterTarget(name, targetType string) {
	if w.Collector != nil {
		w.Collector.RegisterTarget(name, targetType)
	}
}

// UpdateTargetStatus 实现 ICollector 接口
func (w *CollectorWrapper) UpdateTargetStatus(name, status string) {
	if w.Collector != nil {
		w.Collector.UpdateTargetStatus(name, status)
	}
}

// Init 实现 ICollector 接口
func (w *CollectorWrapper) Init(config *entity.MonitorConfig) error {
	if w.Collector != nil {
		return w.Collector.Init(config)
	}
	return nil
}

// Start 实现 ICollector 接口
func (w *CollectorWrapper) Start() error {
	if w.Collector != nil {
		return w.Collector.Start()
	}
	return nil
}

// Stop 实现 ICollector 接口
func (w *CollectorWrapper) Stop() error {
	if w.Collector != nil {
		return w.Collector.Stop()
	}
	return nil
}

// OnMetric 实现 ICollector 接口
func (w *CollectorWrapper) OnMetric(m *entity.SyncMetric) {
	if w.Collector != nil {
		w.Collector.OnMetric(m)
	}
}

// OnRow 实现 ICollector 接口
func (w *CollectorWrapper) OnRow(e *canal.RowsEvent, durationMs int, err error) {
	if w.Collector != nil {
		w.Collector.OnRow(e, durationMs, err)
	}
}

// GetStatus 实现 ICollector 接口
func (w *CollectorWrapper) GetStatus() *entity.ServiceStatus {
	if w.Collector != nil {
		return w.Collector.GetStatus()
	}
	return &entity.ServiceStatus{}
}

// GetTargets 实现 ICollector 接口
func (w *CollectorWrapper) GetTargets() []*entity.TargetStatus {
	if w.Collector != nil {
		return w.Collector.GetTargets()
	}
	return nil
}

// GetEventBuffer 实现 ICollector 接口
func (w *CollectorWrapper) GetEventBuffer() []*entity.SyncEvent {
	if w.Collector != nil {
		return w.Collector.GetEventBuffer()
	}
	return nil
}

// GetErrorBuffer 实现 ICollector 接口
func (w *CollectorWrapper) GetErrorBuffer() []*entity.SyncError {
	if w.Collector != nil {
		return w.Collector.GetErrorBuffer()
	}
	return nil
}

// GetConfig 实现 ICollector 接口
func (w *CollectorWrapper) GetConfig() *entity.MonitorConfig {
	if w.Collector != nil {
		return w.Collector.GetConfig()
	}
	return &entity.MonitorConfig{}
}

// SetEnabled 实现 ICollector 接口
func (w *CollectorWrapper) SetEnabled(enabled bool) {
	if w.Collector != nil {
		w.Collector.SetEnabled(enabled)
	}
}

// RegisterRoutes 注册监控路由
func RegisterRoutes(collector service.ICollector, store service.IStore, version string) {
	ctrl := NewController(collector, store, version)

	// 分组注册 API
	g.Server().Group("/api/monitor", func(group *ghttp.RouterGroup) {
		// 使用响应中间件
		group.Middleware(ghttp.MiddlewareHandlerResponse)

		// 状态相关
		group.Bind(
			ctrl.Status,
			ctrl.Health,
		)

		// 指标相关
		group.Bind(
			ctrl.Metrics,
			ctrl.MetricsHistory,
		)

		// 事件相关
		group.Bind(
			ctrl.Events,
			ctrl.EventStats,
		)

		// 错误相关
		group.Bind(
			ctrl.Errors,
			ctrl.ErrorStats,
		)

		// 位置相关
		group.Bind(
			ctrl.Position,
			ctrl.PositionHistory,
		)

		// 延迟相关
		group.Bind(
			ctrl.Latency,
			ctrl.LatencyHistory,
		)

		// 目标管理
		group.Bind(
			ctrl.Targets,
			ctrl.TargetEnable,
			ctrl.TargetDisable,
		)

		// 配置
		group.Bind(
			ctrl.Config,
			ctrl.UpdateConfig,
		)

		// SSE 实时推送
		group.GET("/status/realtime", ctrl.SSE)
		group.GET("/stream", ctrl.SSE) // 别名，兼容前端
	})
}

// InitMonitor 初始化监控系统
func InitMonitor(config *entity.MonitorConfig, chConfig *entity.ClickHouseConfig, version string) (service.ICollector, service.IStore, error) {
	// 创建存储
	store, err := monitor.NewStore(config, chConfig)
	if err != nil {
		return nil, nil, err
	}

	// 创建采集器
	collector := monitor.NewCollector(config, store)

	// 启动采集器
	if err := collector.Start(); err != nil {
		return nil, nil, err
	}

	// 注册路由
	RegisterRoutes(collector, store, version)

	return collector, store, nil
}
