// =================================================================================
// Monitor Controller - 监控 API 控制器
// =================================================================================

package monitor

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"sync-canal-go/internal/model/entity"
	"sync-canal-go/internal/service"
)

// Controller 监控控制器
type Controller struct {
	collector service.ICollector
	store     service.IStore
	version   string
}

// getStatus 安全获取状态
func (c *Controller) getStatus() *entity.ServiceStatus {
	if c.collector == nil {
		return &entity.ServiceStatus{Status: "uninitialized"}
	}
	return c.collector.GetStatus()
}

// getTargets 安全获取目标列表
func (c *Controller) getTargets() []*entity.TargetStatus {
	if c.collector == nil {
		return []*entity.TargetStatus{}
	}
	return c.collector.GetTargets()
}

// getConfig 安全获取配置
func (c *Controller) getConfig() *entity.MonitorConfig {
	if c.collector == nil {
		return &entity.MonitorConfig{}
	}
	return c.collector.GetConfig()
}

// NewController 创建控制器
func NewController(collector service.ICollector, store service.IStore, version string) *Controller {
	return &Controller{
		collector: collector,
		store:     store,
		version:   version,
	}
}

// StatusReq 状态请求
type StatusReq struct {
	g.Meta `path:"/status" method:"get" tags:"Monitor" summary:"获取服务状态"`
}

// StatusRes 状态响应
type StatusRes struct {
	*entity.ServiceStatus
	Targets []*entity.TargetStatus `json:"targets"` // 目标状态列表
}

// Status 获取服务状态
func (c *Controller) Status(ctx context.Context, req *StatusReq) (res *StatusRes, err error) {
	status := c.getStatus()
	status.Version = c.version

	// 优先使用 collector 的实时 QPS/TPS（由 metricsLoop 计算）
	// 如果实时值为0，则从 ClickHouse 获取最近的事件统计来计算
	if c.store != nil && status.QPS == 0 && status.TPS == 0 {
		now := time.Now()
		// 尝试最近5分钟的数据
		start := now.Add(-5 * time.Minute).Unix()
		end := now.Unix()

		stats, err := c.store.GetEventStats(ctx, start, end)
		if err == nil && stats != nil {
			// 计算最近5分钟的 QPS/TPS
			totalEvents := stats.TotalInsert + stats.TotalUpdate + stats.TotalDelete
			if totalEvents > 0 {
				// 5分钟内的平均值
				status.QPS = float64(totalEvents) / 300.0
				status.TPS = float64(totalEvents) / 300.0
			}
		}
	}

	return &StatusRes{
		ServiceStatus: status,
		Targets:       c.getTargets(),
	}, nil
}

// HealthReq 健康检查请求
type HealthReq struct {
	g.Meta `path:"/health" method:"get" tags:"Monitor" summary:"健康检查"`
}

// HealthRes 健康检查响应
type HealthRes struct {
	*entity.HealthCheck
}

// Health 健康检查
func (c *Controller) Health(ctx context.Context, req *HealthReq) (res *HealthRes, err error) {
	checks := make(map[string]entity.Check)

	// 检查服务状态
	status := c.getStatus()
	serviceStatus := "ok"
	if status.Status != "running" {
		serviceStatus = "error"
	}
	checks["service"] = entity.Check{
		Status:  serviceStatus,
		Message: status.Status,
	}

	// 检查延迟
	delayStatus := "ok"
	if status.DelaySeconds > 60 {
		delayStatus = "warning"
	}
	if status.DelaySeconds > 300 {
		delayStatus = "error"
	}
	checks["latency"] = entity.Check{
		Status:  delayStatus,
		Message: "延迟正常",
	}

	// 检查目标连接
	targetStatus := "ok"
	targets := c.getTargets()
	for _, t := range targets {
		if t.Status == "error" || t.Status == "disconnected" {
			targetStatus = "error"
			break
		}
	}
	checks["targets"] = entity.Check{
		Status:  targetStatus,
		Message: "所有目标正常",
	}

	// 计算整体状态
	overallStatus := "healthy"
	for _, check := range checks {
		if check.Status == "error" {
			overallStatus = "unhealthy"
			break
		}
		if check.Status == "warning" && overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
	}

	return &HealthRes{
		HealthCheck: &entity.HealthCheck{
			Status:      overallStatus,
			Checks:      checks,
			LastChecked: time.Now(),
		},
	}, nil
}

// MetricsReq 指标请求
type MetricsReq struct {
	g.Meta `path:"/metrics" method:"get" tags:"Monitor" summary:"获取当前指标"`
}

// MetricsRes 指标响应
type MetricsRes struct {
	Status    *entity.ServiceStatus `json:"status"`    // 服务状态
	EventStats *entity.EventStats   `json:"eventStats"` // 事件统计
	Targets   []*entity.TargetStatus `json:"targets"`  // 目标状态
}

// Metrics 获取当前指标
func (c *Controller) Metrics(ctx context.Context, req *MetricsReq) (res *MetricsRes, err error) {
	now := time.Now()
	start := now.Add(-24 * time.Hour).Unix()

	eventStats, _ := c.store.GetEventStats(ctx, start, now.Unix())

	status := c.getStatus()
	status.Version = c.version

	// 优先使用 collector 的实时 QPS/TPS
	// 如果实时值为0，从 ClickHouse 获取最近的事件统计来更新
	if c.store != nil && status.QPS == 0 && status.TPS == 0 {
		// 尝试最近5分钟的数据
		recentStart := now.Add(-5 * time.Minute).Unix()
		recentStats, err := c.store.GetEventStats(ctx, recentStart, now.Unix())
		if err == nil && recentStats != nil {
			totalEvents := recentStats.TotalInsert + recentStats.TotalUpdate + recentStats.TotalDelete
			if totalEvents > 0 {
				status.QPS = float64(totalEvents) / 300.0
				status.TPS = float64(totalEvents) / 300.0
			}
		}
	}

	return &MetricsRes{
		Status:     status,
		EventStats: eventStats,
		Targets:    c.getTargets(),
	}, nil
}

// MetricsHistoryReq 指标历史请求
type MetricsHistoryReq struct {
	g.Meta `path:"/metrics/history" method:"get" tags:"Monitor" summary:"获取指标历史"`
	Name   string `json:"name" p:"name" dc:"指标名称"`
	Start  int64  `json:"start" p:"start" dc:"开始时间(Unix时间戳)"`
	End    int64  `json:"end" p:"end" dc:"结束时间(Unix时间戳)"`
}

// MetricsHistoryRes 指标历史响应
type MetricsHistoryRes struct {
	List []*entity.MetricHistory `json:"list"`
}

// MetricsHistory 获取指标历史
func (c *Controller) MetricsHistory(ctx context.Context, req *MetricsHistoryReq) (res *MetricsHistoryRes, err error) {
	if req.End == 0 {
		req.End = time.Now().Unix()
	}
	if req.Start == 0 {
		req.Start = req.End - 3600 // 默认1小时
	}

	list, err := c.store.GetMetricHistory(ctx, req.Name, req.Start, req.End)
	if err != nil {
		return nil, err
	}

	return &MetricsHistoryRes{List: list}, nil
}

// EventsReq 事件列表请求
type EventsReq struct {
	g.Meta     `path:"/events" method:"get" tags:"Monitor" summary:"获取事件列表"`
	TargetName string `json:"targetName" p:"targetName" dc:"目标名称"`
	TableName  string `json:"tableName" p:"tableName" dc:"表名"`
	EventType  string `json:"eventType" p:"eventType" dc:"事件类型(INSERT/UPDATE/DELETE)"`
	StartTime  int64  `json:"startTime" p:"startTime" dc:"开始时间(Unix时间戳)"`
	EndTime    int64  `json:"endTime" p:"endTime" dc:"结束时间(Unix时间戳)"`
	Page       int    `json:"page" p:"page" dc:"页码" d:"1"`
	PageSize   int    `json:"pageSize" p:"pageSize" dc:"每页数量" d:"20"`
}

// EventsRes 事件列表响应
type EventsRes struct {
	List  []*entity.SyncEvent `json:"list"`
	Total int64               `json:"total"`
}

// Events 获取事件列表
func (c *Controller) Events(ctx context.Context, req *EventsReq) (res *EventsRes, err error) {
	query := &service.EventQuery{
		TargetName: req.TargetName,
		TableName:  req.TableName,
		EventType:  req.EventType,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}

	list, total, err := c.store.GetEvents(ctx, *query)
	if err != nil {
		return nil, err
	}

	return &EventsRes{List: list, Total: total}, nil
}

// EventStatsReq 事件统计请求
type EventStatsReq struct {
	g.Meta `path:"/events/stats" method:"get" tags:"Monitor" summary:"获取事件统计"`
	Start  int64 `json:"start" p:"start" dc:"开始时间(Unix时间戳)"`
	End    int64 `json:"end" p:"end" dc:"结束时间(Unix时间戳)"`
}

// EventStatsRes 事件统计响应
type EventStatsRes struct {
	*entity.EventStats
}

// EventStats 获取事件统计
func (c *Controller) EventStats(ctx context.Context, req *EventStatsReq) (res *EventStatsRes, err error) {
	if req.End == 0 {
		req.End = time.Now().Unix()
	}
	if req.Start == 0 {
		req.Start = req.End - 86400*7 // 默认7天
	}

	stats, err := c.store.GetEventStats(ctx, req.Start, req.End)
	if err != nil {
		return nil, err
	}

	return &EventStatsRes{EventStats: stats}, nil
}

// ErrorsReq 错误列表请求
type ErrorsReq struct {
	g.Meta     `path:"/errors" method:"get" tags:"Monitor" summary:"获取错误列表"`
	Level      string `json:"level" p:"level" dc:"错误级别(ERROR/WARNING)"`
	TargetName string `json:"targetName" p:"targetName" dc:"目标名称"`
	TableName  string `json:"tableName" p:"tableName" dc:"表名"`
	Keyword    string `json:"keyword" p:"keyword" dc:"关键词搜索"`
	StartTime  int64  `json:"startTime" p:"startTime" dc:"开始时间(Unix时间戳)"`
	EndTime    int64  `json:"endTime" p:"endTime" dc:"结束时间(Unix时间戳)"`
	Page       int    `json:"page" p:"page" dc:"页码" d:"1"`
	PageSize   int    `json:"pageSize" p:"pageSize" dc:"每页数量" d:"20"`
}

// ErrorsRes 错误列表响应
type ErrorsRes struct {
	List  []*entity.SyncError `json:"list"`
	Total int64               `json:"total"`
}

// Errors 获取错误列表
func (c *Controller) Errors(ctx context.Context, req *ErrorsReq) (res *ErrorsRes, err error) {
	query := &service.ErrorQuery{
		Level:      req.Level,
		TargetName: req.TargetName,
		TableName:  req.TableName,
		Keyword:    req.Keyword,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}

	list, total, err := c.store.GetErrors(ctx, *query)
	if err != nil {
		return nil, err
	}

	return &ErrorsRes{List: list, Total: total}, nil
}

// ErrorStatsReq 错误统计请求
type ErrorStatsReq struct {
	g.Meta `path:"/errors/stats" method:"get" tags:"Monitor" summary:"获取错误统计"`
	Start  int64 `json:"start" p:"start" dc:"开始时间(Unix时间戳)"`
	End    int64 `json:"end" p:"end" dc:"结束时间(Unix时间戳)"`
}

// ErrorStatsRes 错误统计响应
type ErrorStatsRes struct {
	*entity.ErrorStats
}

// ErrorStats 获取错误统计
func (c *Controller) ErrorStats(ctx context.Context, req *ErrorStatsReq) (res *ErrorStatsRes, err error) {
	if req.End == 0 {
		req.End = time.Now().Unix()
	}
	if req.Start == 0 {
		req.Start = req.End - 86400*7 // 默认7天
	}

	stats, err := c.store.GetErrorStats(ctx, req.Start, req.End)
	if err != nil {
		return nil, err
	}

	return &ErrorStatsRes{ErrorStats: stats}, nil
}

// PositionReq 位置请求
type PositionReq struct {
	g.Meta `path:"/position" method:"get" tags:"Monitor" summary:"获取当前同步位置"`
}

// PositionRes 位置响应
type PositionRes struct {
	*entity.ServiceStatus
}

// Position 获取当前同步位置
func (c *Controller) Position(ctx context.Context, req *PositionReq) (res *PositionRes, err error) {
	status := c.getStatus()
	return &PositionRes{ServiceStatus: status}, nil
}

// PositionHistoryReq 位置历史请求
type PositionHistoryReq struct {
	g.Meta `path:"/position/history" method:"get" tags:"Monitor" summary:"获取位置历史"`
	Start  int64 `json:"start" p:"start" dc:"开始时间(Unix时间戳)"`
	End    int64 `json:"end" p:"end" dc:"结束时间(Unix时间戳)"`
}

// PositionHistoryRes 位置历史响应
type PositionHistoryRes struct {
	List []*entity.PositionHistory `json:"list"`
}

// PositionHistory 获取位置历史
func (c *Controller) PositionHistory(ctx context.Context, req *PositionHistoryReq) (res *PositionHistoryRes, err error) {
	if req.End == 0 {
		req.End = time.Now().Unix()
	}
	if req.Start == 0 {
		req.Start = req.End - 3600
	}

	// 获取第一个目标的历史
	targets := c.getTargets()
	if len(targets) == 0 {
		return &PositionHistoryRes{}, nil
	}

	list, err := c.store.GetPositionHistory(ctx, targets[0].Name, req.Start, req.End)
	if err != nil {
		return nil, err
	}

	return &PositionHistoryRes{List: list}, nil
}

// LatencyReq 延迟请求
type LatencyReq struct {
	g.Meta     `path:"/latency" method:"get" tags:"Monitor" summary:"获取当前延迟"`
	StartTime  int64  `json:"startTime" p:"startTime" dc:"开始时间(Unix时间戳)"`
	EndTime    int64  `json:"endTime" p:"endTime" dc:"结束时间(Unix时间戳)"`
	TargetName string `json:"targetName" p:"targetName" dc:"目标名称"`
}

// LatencyRes 延迟响应
type LatencyRes struct {
	*entity.LatencyStats
}

// Latency 获取当前延迟
func (c *Controller) Latency(ctx context.Context, req *LatencyReq) (res *LatencyRes, err error) {
	stats := &entity.LatencyStats{}

	// 从 collector 获取当前延迟
	if c.collector != nil {
		status := c.collector.GetStatus()
		if status != nil {
			stats.CurrentDelay = status.DelaySeconds
		}
	}

	if c.store != nil {
		// 确定时间范围
		end := req.EndTime
		if end == 0 {
			end = time.Now().Unix()
		}
		start := req.StartTime
		if start == 0 {
			start = end - 3600 // 默认1小时
		}

		chStats, err := c.store.GetLatencyStats(ctx, start, end)
		if err == nil && chStats != nil {
			stats.AvgDelay = chStats.AvgDelay
			stats.MaxDelay = chStats.MaxDelay
			stats.MinDelay = chStats.MinDelay
			stats.CurrentDelay = chStats.CurrentDelay
			stats.P95Delay = chStats.P95Delay
			stats.P99Delay = chStats.P99Delay
		}
	}

	return &LatencyRes{LatencyStats: stats}, nil
}

// LatencyHistoryReq 延迟历史请求
type LatencyHistoryReq struct {
	g.Meta `path:"/latency/history" method:"get" tags:"Monitor" summary:"获取延迟历史"`
	Start  int64 `json:"start" p:"start" dc:"开始时间(Unix时间戳)"`
	End    int64 `json:"end" p:"end" dc:"结束时间(Unix时间戳)"`
}

// LatencyHistoryRes 延迟历史响应
type LatencyHistoryRes struct {
	List []*entity.MetricHistory `json:"list"`
}

// LatencyHistory 获取延迟历史
func (c *Controller) LatencyHistory(ctx context.Context, req *LatencyHistoryReq) (res *LatencyHistoryRes, err error) {
	if req.End == 0 {
		req.End = time.Now().Unix()
	}
	if req.Start == 0 {
		req.Start = req.End - 3600
	}

	// 从 sync_positions 表获取延迟历史
	list, err := c.store.GetLatencyHistory(ctx, req.Start, req.End)
	if err != nil {
		return nil, err
	}

	return &LatencyHistoryRes{List: list}, nil
}

// TargetsReq 目标列表请求
type TargetsReq struct {
	g.Meta `path:"/targets" method:"get" tags:"Monitor" summary:"获取目标列表"`
}

// TargetsRes 目标列表响应
type TargetsRes struct {
	List []*entity.TargetStatus `json:"list"`
}

// Targets 获取目标列表
func (c *Controller) Targets(ctx context.Context, req *TargetsReq) (res *TargetsRes, err error) {
	// 首先从 collector 获取内存中的目标
	targets := c.getTargets()

	// 如果有存储，从 ClickHouse 补充统计信息
	if c.store != nil && len(targets) > 0 {
		now := time.Now()
		start := now.Add(-7 * 24 * time.Hour).Unix() // 最近7天
		end := now.Unix()

		for _, t := range targets {
			stats, err := c.store.GetTargetStats(ctx, t.Name, start, end)
			if err == nil && stats != nil {
				t.TotalEvents = stats.TotalEvents
				t.TotalErrors = stats.TotalErrors
				t.LastSyncTime = stats.LastSyncTime
			}
		}
	}

	// 如果内存中没有目标，尝试从 ClickHouse 获取
	if len(targets) == 0 && c.store != nil {
		targetNames, err := c.store.GetTargetsFromEvents(ctx)
		if err == nil && len(targetNames) > 0 {
			now := time.Now()
			start := now.Add(-7 * 24 * time.Hour).Unix() // 最近7天
			end := now.Unix()

			for _, name := range targetNames {
				stats, err := c.store.GetTargetStats(ctx, name, start, end)
				if err == nil && stats != nil {
					targets = append(targets, stats)
				}
			}
		}
	}

	return &TargetsRes{List: targets}, nil
}

// TargetEnableReq 启用目标请求
type TargetEnableReq struct {
	g.Meta `path:"/targets/{name}/enable" method:"post" tags:"Monitor" summary:"启用目标"`
	Name   string `json:"name" dc:"目标名称"`
}

// TargetEnableRes 启用目标响应
type TargetEnableRes struct {
	Success bool `json:"success"`
}

// TargetEnable 启用目标
func (c *Controller) TargetEnable(ctx context.Context, req *TargetEnableReq) (res *TargetEnableRes, err error) {
	if c.collector != nil {
		c.collector.UpdateTargetStatus(req.Name, "connected")
	}
	return &TargetEnableRes{Success: true}, nil
}

// TargetDisableReq 禁用目标请求
type TargetDisableReq struct {
	g.Meta `path:"/targets/{name}/disable" method:"post" tags:"Monitor" summary:"禁用目标"`
	Name   string `json:"name" dc:"目标名称"`
}

// TargetDisableRes 禁用目标响应
type TargetDisableRes struct {
	Success bool `json:"success"`
}

// TargetDisable 禁用目标
func (c *Controller) TargetDisable(ctx context.Context, req *TargetDisableReq) (res *TargetDisableRes, err error) {
	if c.collector != nil {
		c.collector.UpdateTargetStatus(req.Name, "disabled")
	}
	return &TargetDisableRes{Success: true}, nil
}

// ConfigReq 配置请求
type ConfigReq struct {
	g.Meta `path:"/config" method:"get" tags:"Monitor" summary:"获取当前配置"`
}

// ConfigRes 配置响应
type ConfigRes struct {
	*entity.MonitorConfig
	ClickHouse *ClickHouseConfigRes `json:"clickhouse"`
}

// ClickHouseConfigRes ClickHouse 配置响应
type ClickHouseConfigRes struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
}

// Config 获取当前配置
func (c *Controller) Config(ctx context.Context, req *ConfigReq) (res *ConfigRes, err error) {
	// 优先从 collector 获取运行时配置
	monitorConfig := c.getConfig()

	// 如果 collector 没有配置，从配置文件读取
	if monitorConfig == nil {
		cfg := g.Cfg()
		monitorConfig = &entity.MonitorConfig{
			Enabled:       cfg.MustGet(ctx, "monitor.enabled", true).Bool(),
			HistoryDays:   cfg.MustGet(ctx, "monitor.historyDays", 30).Int(),
			CollectPeriod: cfg.MustGet(ctx, "monitor.collectPeriod", 10).Int(),
			MaxEvents:     cfg.MustGet(ctx, "monitor.maxEvents", 10000).Int(),
			MaxErrors:     cfg.MustGet(ctx, "monitor.maxErrors", 1000).Int(),
		}
	}

	// ClickHouse 配置从配置文件读取
	cfg := g.Cfg()
	chConfig := &ClickHouseConfigRes{
		Host:     cfg.MustGet(ctx, "monitor.clickhouse.host", "127.0.0.1").String(),
		Port:     cfg.MustGet(ctx, "monitor.clickhouse.port", 8124).Int(),
		Database: cfg.MustGet(ctx, "monitor.clickhouse.database", "sync_monitor").String(),
	}

	return &ConfigRes{
		MonitorConfig: monitorConfig,
		ClickHouse:    chConfig,
	}, nil
}

// UpdateConfigReq 更新配置请求
type UpdateConfigReq struct {
	g.Meta       `path:"/config" method:"post" tags:"Monitor" summary:"更新配置"`
	*entity.MonitorConfig
	ClickHouse *ClickHouseConfigRes `json:"clickhouse"`
}

// UpdateConfigRes 更新配置响应
type UpdateConfigRes struct {
	Success bool `json:"success"`
}

// UpdateConfig 更新配置
func (c *Controller) UpdateConfig(ctx context.Context, req *UpdateConfigReq) (res *UpdateConfigRes, err error) {
	// 更新 collector 配置
	if c.collector != nil {
		monitorConfig := &entity.MonitorConfig{
			Enabled:       req.Enabled,
			HistoryDays:   req.HistoryDays,
			CollectPeriod: req.CollectPeriod,
			MaxEvents:     req.MaxEvents,
			MaxErrors:     req.MaxErrors,
		}
		if err := c.collector.Init(monitorConfig); err != nil {
			return &UpdateConfigRes{Success: false}, err
		}
	}

	g.Log().Infof(ctx, "Monitor config updated: enabled=%v, historyDays=%d, collectPeriod=%d, maxEvents=%d, maxErrors=%d",
		req.Enabled, req.HistoryDays, req.CollectPeriod, req.MaxEvents, req.MaxErrors)

	return &UpdateConfigRes{Success: true}, nil
}

// SSE 实时状态推送
func (c *Controller) SSE(r *ghttp.Request) {
	r.Response.Header().Set("Content-Type", "text/event-stream")
	r.Response.Header().Set("Cache-Control", "no-cache")
	r.Response.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			status := c.getStatus()
			status.Version = c.version

			// 发送 SSE 事件
			b, _ := json.Marshal(status)
			r.Response.Writef("data: %s\n\n", string(b))
			r.Response.Flush()
		}
	}
}
