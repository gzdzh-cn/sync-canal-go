// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package service

import (
	"context"
	"os"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"

	"sync-canal-go/internal/model/entity"
)

// SyncTarget 同步目标接口
type SyncTarget interface {
	// Connect 连接目标
	Connect(ctx context.Context) error
	// Close 关闭连接
	Close() error
	// OnRow 处理行变更事件
	OnRow(e *canal.RowsEvent) error
	// OnDDL 处理 DDL 事件
	OnDDL(header *replication.EventHeader, nextPos mysql.Position, queryEvent *replication.QueryEvent) error
	// Start 启动目标的后台任务（如定时清理）
	Start()
	// String 返回目标名称
	String() string
}

// ISync 同步服务接口
type ISync interface {
	// InitTimezone 初始化时区
	InitTimezone()
	// LoadConfig 加载配置文件
	LoadConfig(ctx context.Context) (*entity.SyncConfig, *entity.CanalConfig, error)
	// CreateCanal 创建 Canal
	CreateCanal(syncConfig *entity.SyncConfig, canalConfig *entity.CanalConfig) (*canal.Canal, error)
	// CreateTargets 根据配置创建所有同步目标
	CreateTargets(config *entity.SyncConfig) ([]SyncTarget, error)
	// WaitForSignal 等待退出信号
	WaitForSignal() os.Signal
	// LogShutdownInfo 记录关闭信息
	LogShutdownInfo(ctx context.Context, c *canal.Canal)
}
