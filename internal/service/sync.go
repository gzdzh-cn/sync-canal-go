// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"os"
	"sync-canal-go/internal/model/entity"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
)

type (
	ICanalSync interface {
		// Start 启动 Canal 同步服务
		Start(ctx context.Context)
	}
	IClickHouseTarget interface {
		// Connect 连接 ClickHouse
		Connect(ctx context.Context) error
		// Close 关闭连接
		Close() error
		// OnRow 处理行变更事件
		OnRow(e *canal.RowsEvent) error
		// OnDDL 处理 DDL 事件
		OnDDL(header *replication.EventHeader, nextPos mysql.Position, queryEvent *replication.QueryEvent) error
		// Start 启动定时清理任务
		Start()
		// String 返回目标名称
		String() string
	}
	IMultiHandler interface {
		// OnRow 将行变更事件分发到所有目标
		OnRow(e *canal.RowsEvent) error
		// OnRotate 处理 binlog 轮转
		OnRotate(header *replication.EventHeader, e *replication.RotateEvent) error
		// OnDDL 将 DDL 事件分发到所有目标
		OnDDL(header *replication.EventHeader, nextPos mysql.Position, queryEvent *replication.QueryEvent) error
		// OnXID 处理事务提交
		OnXID(header *replication.EventHeader, nextPos mysql.Position) error
		// StartTargets 启动所有目标的后台任务
		StartTargets()
		// CloseTargets 关闭所有目标
		CloseTargets()
		// String 返回处理器名称
		String() string
	}
	ISync interface {
		// InitTimezone 初始化时区
		InitTimezone()
		// LoadConfig 加载配置文件
		LoadConfig(ctx context.Context) (*entity.SyncConfig, *entity.CanalConfig, error)
		// CreateCanal 创建 Canal
		CreateCanal(syncConfig *entity.SyncConfig, canalConfig *entity.CanalConfig) (*canal.Canal, error)
		// WaitForSignal 等待退出信号
		WaitForSignal() os.Signal
		// LogShutdownInfo 记录关闭信息
		LogShutdownInfo(ctx context.Context, c *canal.Canal)
		// CreateTargets 根据配置创建所有同步目标
		CreateTargets(config *entity.SyncConfig) ([]SyncTarget, error)
	}
)

var (
	localCanalSync        ICanalSync
	localClickHouseTarget IClickHouseTarget
	localMultiHandler     IMultiHandler
	localSync             ISync
)

func CanalSync() ICanalSync {
	if localCanalSync == nil {
		panic("implement not found for interface ICanalSync, forgot register?")
	}
	return localCanalSync
}

func RegisterCanalSync(i ICanalSync) {
	localCanalSync = i
}

func ClickHouseTarget() IClickHouseTarget {
	if localClickHouseTarget == nil {
		panic("implement not found for interface IClickHouseTarget, forgot register?")
	}
	return localClickHouseTarget
}

func RegisterClickHouseTarget(i IClickHouseTarget) {
	localClickHouseTarget = i
}

func MultiHandler() IMultiHandler {
	if localMultiHandler == nil {
		panic("implement not found for interface IMultiHandler, forgot register?")
	}
	return localMultiHandler
}

func RegisterMultiHandler(i IMultiHandler) {
	localMultiHandler = i
}

func Sync() ISync {
	if localSync == nil {
		panic("implement not found for interface ISync, forgot register?")
	}
	return localSync
}

func RegisterSync(i ISync) {
	localSync = i
}
