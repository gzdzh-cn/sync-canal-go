// =================================================================================
// SyncTarget 同步目标接口定义
// =================================================================================

package service

import (
	"context"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
)

// SyncTarget 同步目标接口
type SyncTarget interface {
	Connect(ctx context.Context) error
	Close() error
	OnRow(e *canal.RowsEvent) error
	OnDDL(header *replication.EventHeader, nextPos mysql.Position, queryEvent *replication.QueryEvent) error
	Start()
	String() string
}
