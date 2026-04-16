// =================================================================================
// Canal Sync Service - Canal 同步服务
// =================================================================================

package sync

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"sync-canal-go/internal/service"
)

// sCanalSync Canal 同步服务
type sCanalSync struct {
	collector service.ICollector
}

// NewCanalSync 创建 Canal 同步服务
func NewCanalSync(collector service.ICollector) *sCanalSync {
	return &sCanalSync{collector: collector}
}

// Start 启动 Canal 同步服务
func (s *sCanalSync) Start(ctx context.Context) {
	// 初始化时区
	service.Sync().InitTimezone()

	// 加载配置
	syncConfig, canalConfig, err := service.Sync().LoadConfig(ctx)
	if err != nil {
		g.Log().Fatal(ctx, "Load config failed:", err)
	}

	// 创建 Canal
	c, err := service.Sync().CreateCanal(syncConfig, canalConfig)
	if err != nil {
		g.Log().Fatal(ctx, "Create canal failed:", err)
	}
	defer c.Close()

	g.Log().Infof(ctx, "Canal created, serverId: %d", canalConfig.ServerId)

	// 创建所有同步目标
	targets, err := service.Sync().CreateTargets(syncConfig)
	if err != nil {
		g.Log().Fatal(ctx, "Create targets failed:", err)
	}
	defer func() {
		for _, t := range targets {
			_ = t.Close()
		}
	}()

	// 设置事件处理器（MultiHandler 广播到所有目标）
	handler := NewMultiHandler(targets, s.collector)
	c.SetEventHandler(handler)

	// 启动所有目标的后台任务
	handler.StartTargets()

	// 获取当前 binlog 位置
	pos, err := c.GetMasterPos()
	if err != nil {
		g.Log().Fatal(ctx, "Get master position failed:", err)
	}

	g.Log().Infof(ctx, "Start syncing from %s:%d", pos.Name, pos.Pos)

	// 启动同步（阻塞）
	if err := c.RunFrom(pos); err != nil {
		g.Log().Fatal(ctx, "Run canal failed:", err)
	}
}
