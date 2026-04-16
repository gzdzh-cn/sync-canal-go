// =================================================================================
// Monitor Service Interface
// =================================================================================

package service

import (
	"context"

	"github.com/go-mysql-org/go-mysql/canal"

	"sync-canal-go/internal/model/entity"
)

// IMonitor 监控服务接口
type IMonitor interface {
	// 状态相关
	GetStatus(ctx context.Context) *entity.ServiceStatus
	GetHealth(ctx context.Context) *entity.HealthCheck
	GetTargetStatus(ctx context.Context, targetName string) *entity.TargetStatus
	GetAllTargetStatus(ctx context.Context) []*entity.TargetStatus

	// 指标相关
	GetMetrics(ctx context.Context) []*entity.SyncMetric
	GetMetricHistory(ctx context.Context, name string, start, end int64) []*entity.MetricHistory
	GetEventStats(ctx context.Context, start, end int64) *entity.EventStats
	GetLatencyStats(ctx context.Context, start, end int64) *entity.LatencyStats
	GetErrorStats(ctx context.Context, start, end int64) *entity.ErrorStats

	// 事件相关
	GetEvents(ctx context.Context, query *EventQuery) []*entity.SyncEvent
	GetEventByID(ctx context.Context, id string) *entity.SyncEvent

	// 错误相关
	GetErrors(ctx context.Context, query *ErrorQuery) []*entity.SyncError
	GetRecentErrors(ctx context.Context, limit int) []*entity.SyncError

	// 位置相关
	GetPosition(ctx context.Context) *entity.SyncPosition
	GetPositionHistory(ctx context.Context, start, end int64) []*entity.PositionHistory
	SetPosition(ctx context.Context, pos *entity.SyncPosition) error

	// 延迟相关
	GetLatency(ctx context.Context) *entity.LatencyStats
	GetLatencyHistory(ctx context.Context, start, end int64) []*entity.MetricHistory

	// 目标管理
	EnableTarget(ctx context.Context, name string) error
	DisableTarget(ctx context.Context, name string) error

	// 采集相关
	CollectEvent(e *canal.RowsEvent, durationMs int, err error)
	CollectPosition(file string, pos uint32, delay int64)
	CollectError(level, target, table, message string, err error)
}

// EventQuery 事件查询条件
type EventQuery struct {
	TargetName string `json:"targetName"` // 目标名称
	TableName  string `json:"tableName"`  // 表名
	EventType  string `json:"eventType"`  // 事件类型
	Success    *bool  `json:"success"`    // 是否成功
	StartTime  int64  `json:"startTime"`  // 开始时间
	EndTime    int64  `json:"endTime"`    // 结束时间
	Page       int    `json:"page"`       // 页码
	PageSize   int    `json:"pageSize"`   // 每页数量
}

// ErrorQuery 错误查询条件
type ErrorQuery struct {
	Level      string `json:"level"`      // 错误级别
	TargetName string `json:"targetName"` // 目标名称
	TableName  string `json:"tableName"`  // 表名
	Keyword    string `json:"keyword"`    // 关键词搜索
	StartTime  int64  `json:"startTime"`  // 开始时间
	EndTime    int64  `json:"endTime"`    // 结束时间
	Page       int    `json:"page"`       // 页码
	PageSize   int    `json:"pageSize"`   // 每页数量
}

// ICollector 采集器接口
type ICollector interface {
	// 初始化
	Init(config *entity.MonitorConfig) error

	// 采集事件
	OnEvent(e *entity.SyncEvent)

	// 采集位置
	OnPosition(p *entity.SyncPosition)

	// 采集错误
	OnError(err *entity.SyncError)

	// 采集指标
	OnMetric(m *entity.SyncMetric)

	// 启动/停止
	Start() error
	Stop() error
}

// IStore 存储接口
type IStore interface {
	// 事件存储
	SaveEvent(ctx context.Context, e *entity.SyncEvent) error
	SaveEvents(ctx context.Context, events []*entity.SyncEvent) error
	GetEvents(ctx context.Context, query *EventQuery) ([]*entity.SyncEvent, int64, error)

	// 错误存储
	SaveError(ctx context.Context, err *entity.SyncError) error
	GetErrors(ctx context.Context, query *ErrorQuery) ([]*entity.SyncError, int64, error)

	// 位置存储
	SavePosition(ctx context.Context, p *entity.SyncPosition) error
	GetPosition(ctx context.Context, targetName string) (*entity.SyncPosition, error)
	GetPositionHistory(ctx context.Context, targetName string, start, end int64) ([]*entity.PositionHistory, error)

	// 指标存储
	SaveMetric(ctx context.Context, m *entity.SyncMetric) error
	SaveMetrics(ctx context.Context, metrics []*entity.SyncMetric) error
	GetMetricHistory(ctx context.Context, name string, start, end int64) ([]*entity.MetricHistory, error)
	GetLatencyHistory(ctx context.Context, start, end int64) ([]*entity.MetricHistory, error)

	// 统计查询
	GetEventStats(ctx context.Context, start, end int64) (*entity.EventStats, error)
	GetLatencyStats(ctx context.Context, start, end int64) (*entity.LatencyStats, error)
	GetErrorStats(ctx context.Context, start, end int64) (*entity.ErrorStats, error)

	// 目标统计
	GetTargetsFromEvents(ctx context.Context) ([]string, error)
	GetTargetStats(ctx context.Context, targetName string, start, end int64) (*entity.TargetStatus, error)

	// 清理
	Cleanup(ctx context.Context, before int64) error
}
