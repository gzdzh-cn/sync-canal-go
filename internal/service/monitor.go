// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"sync-canal-go/internal/model/entity"

	"github.com/go-mysql-org/go-mysql/canal"
)

type (
	ICollector interface {
		// Init 初始化/更新配置
		Init(config *entity.MonitorConfig) error
		// Start 启动采集器
		Start() error
		// Stop 停止采集器
		Stop() error
		// OnEvent 处理事件
		OnEvent(e *entity.SyncEvent)
		// OnPosition 处理位置更新
		OnPosition(p *entity.SyncPosition)
		// OnError 处理错误
		OnError(err *entity.SyncError)
		// OnMetric 处理指标
		OnMetric(m *entity.SyncMetric)
		// OnRow 处理 canal 行事件
		OnRow(e *canal.RowsEvent, durationMs int, err error)
		// RegisterTarget 注册目标
		RegisterTarget(name string, targetType string)
		// UpdateTargetStatus 更新目标状态
		UpdateTargetStatus(name string, status string)
		// GetStatus 获取状态
		GetStatus() *entity.ServiceStatus
		// GetTargets 获取所有目标状态
		GetTargets() []*entity.TargetStatus
		// GetEventBuffer 获取事件缓冲区
		GetEventBuffer() []*entity.SyncEvent
		// GetErrorBuffer 获取错误缓冲区
		GetErrorBuffer() []*entity.SyncError
		// GetConfig 获取当前配置
		GetConfig() *entity.MonitorConfig
		// SetEnabled 设置监控启用状态
		SetEnabled(enabled bool)
	}
	IStore interface {
		// Close 关闭连接
		Close() error
		// SaveEvent 保存事件
		SaveEvent(ctx context.Context, e *entity.SyncEvent) error
		// SaveEvents 批量保存事件
		SaveEvents(ctx context.Context, events []*entity.SyncEvent) error
		// GetEvents 查询事件列表
		GetEvents(ctx context.Context, query EventQuery) ([]*entity.SyncEvent, int64, error)
		// SaveError 保存错误
		SaveError(ctx context.Context, err *entity.SyncError) error
		// GetErrors 查询错误列表
		GetErrors(ctx context.Context, query ErrorQuery) ([]*entity.SyncError, int64, error)
		// SavePosition 保存位置
		SavePosition(ctx context.Context, p *entity.SyncPosition) error
		// GetPosition 获取当前位置
		GetPosition(ctx context.Context, targetName string) (*entity.SyncPosition, error)
		// GetPositionHistory 获取位置历史
		GetPositionHistory(ctx context.Context, targetName string, start int64, end int64) ([]*entity.PositionHistory, error)
		// SaveMetric 保存指标
		SaveMetric(ctx context.Context, m *entity.SyncMetric) error
		// SaveMetrics 批量保存指标
		SaveMetrics(ctx context.Context, metrics []*entity.SyncMetric) error
		// GetMetricHistory 获取指标历史
		GetMetricHistory(ctx context.Context, name string, start int64, end int64) ([]*entity.MetricHistory, error)
		// GetEventStats 获取事件统计
		GetEventStats(ctx context.Context, start int64, end int64) (*entity.EventStats, error)
		// GetLatencyStats 获取延迟统计
		GetLatencyStats(ctx context.Context, start int64, end int64) (*entity.LatencyStats, error)
		// GetLatencyHistory 获取延迟历史
		GetLatencyHistory(ctx context.Context, start int64, end int64) ([]*entity.MetricHistory, error)
		// GetErrorStats 获取错误统计
		GetErrorStats(ctx context.Context, start int64, end int64) (*entity.ErrorStats, error)
		// GetTargetsFromEvents 从事件数据中获取目标列表
		GetTargetsFromEvents(ctx context.Context) ([]string, error)
		// GetTargetStats 获取目标统计信息
		GetTargetStats(ctx context.Context, targetName string, start int64, end int64) (*entity.TargetStatus, error)
		// Cleanup 清理过期数据
		Cleanup(ctx context.Context, before int64) error
	}
)

var (
	localCollector ICollector
	localStore     IStore
)

func Collector() ICollector {
	if localCollector == nil {
		panic("implement not found for interface ICollector, forgot register?")
	}
	return localCollector
}

func RegisterCollector(i ICollector) {
	localCollector = i
}

func Store() IStore {
	if localStore == nil {
		panic("implement not found for interface IStore, forgot register?")
	}
	return localStore
}

func RegisterStore(i IStore) {
	localStore = i
}
