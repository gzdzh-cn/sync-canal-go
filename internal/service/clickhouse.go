// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package service

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"sync-canal-go/internal/model/entity"
)

// IClickHouse ClickHouse 同步目标接口
type IClickHouse interface {
	// NewClickHouseTarget 创建 ClickHouse 目标
	NewClickHouseTarget(tc *entity.TargetConfig, sync *entity.SyncConfig) (SyncTarget, error)
	// Connect 连接 ClickHouse
	Connect(ctx context.Context, ch *entity.ClickHouseConfig) (driver.Conn, error)
	// HandleInsert 处理 INSERT 事件
	HandleInsert(ctx context.Context, conn driver.Conn, database string, tableName string, rows [][]any) error
	// HandleUpdate 处理 UPDATE 事件
	HandleUpdate(ctx context.Context, conn driver.Conn, database string, tableName string, rows [][]any) error
	// HandleDelete 处理 DELETE 事件
	HandleDelete(ctx context.Context, conn driver.Conn, database string, tableName string, rows [][]any, pkIndex int) error
	// OptimizeTable 执行表优化
	OptimizeTable(ctx context.Context, conn driver.Conn, database string, tables []string) error
}
