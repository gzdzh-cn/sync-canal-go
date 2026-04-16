// =================================================================================
// App - 应用启动与初始化
// =================================================================================

package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gres"

	ctlmonitor "sync-canal-go/internal/controller/monitor"
	logicmonitor "sync-canal-go/internal/logic/monitor"
	"sync-canal-go/internal/logic/sync"
	"sync-canal-go/internal/service"
)

// sApp 应用实例
type sApp struct {
	collector   service.ICollector
	canalCancel context.CancelFunc
}

func init() {
	service.RegisterApp(NewApp())
}

// NewApp 创建应用实例
func NewApp() service.IApp {
	return &sApp{}
}

// SetupStaticFiles 配置静态文件服务（前端 UI）
func (a *sApp) SetupStaticFiles() {
	s := g.Server()

	// 使用 BindHandler 处理静态文件和 SPA 路由（优先级低于路由组）
	s.BindHandler("/*", func(r *ghttp.Request) {
		// 尝试从资源中读取静态文件
		filePath := r.URL.Path
		if filePath == "/" {
			filePath = "/index.html"
		}

		// 从打包的资源中读取
		file := gres.Get("resource/public" + filePath)
		if file != nil {
			r.Response.Write(file.Content())
			return
		}

		// SPA 应用：所有未匹配的路由返回 index.html
		indexFile := gres.Get("resource/public/index.html")
		if indexFile != nil {
			r.Response.Write(indexFile.Content())
			return
		}

		// 没有找到静态文件，返回 404
		r.Response.WriteStatus(404)
	})
}

// InitMonitor 初始化监控系统
func (a *sApp) InitMonitor(ctx context.Context) {
	monitorCfg, chCfg := logicmonitor.LoadConfig(ctx)
	if !monitorCfg.Enabled {
		return
	}

	collector, _, err := ctlmonitor.InitMonitor(monitorCfg, chCfg, "1.0.0")
	if err != nil {
		g.Log().Warningf(ctx, "Init monitor failed: %v, monitoring disabled", err)
		return
	}

	g.Log().Info(ctx, "Monitor system initialized successfully")
	a.collector = collector
}

// StartCanalSyncAsync 异步启动 Canal 同步服务
func (a *sApp) StartCanalSyncAsync() {
	// 创建独立的 context，用于控制 Canal 生命周期
	canalCtx, cancel := context.WithCancel(context.Background())
	a.canalCancel = cancel

	go func() {
		// 捕获协程 panic
		defer func() {
			if r := recover(); r != nil {
				g.Log().Fatalf(context.Background(), "Canal sync panic: %v", r)
			}
		}()

		canalSync := sync.NewCanalSync(a.collector)
		canalSync.Start(canalCtx)
	}()

	g.Log().Info(context.Background(), "Canal sync service started in background")
}

// Stop 优雅停止服务
func (a *sApp) Stop() {
	g.Log().Info(context.Background(), "Stopping app...")

	// 停止 Canal 同步
	if a.canalCancel != nil {
		a.canalCancel()
		g.Log().Info(context.Background(), "Canal sync stopped")
	}

	// 停止监控采集器
	if a.collector != nil {
		if err := a.collector.Stop(); err != nil {
			g.Log().Warningf(context.Background(), "Stop collector failed: %v", err)
		}
	}

	g.Log().Info(context.Background(), "App stopped")
}

// WaitForShutdown 等待关闭信号并优雅关闭
func (a *sApp) WaitForShutdown() {
	defer func() {
		if r := recover(); r != nil {
			g.Log().Fatalf(context.Background(), "Graceful shutdown panic: %v", r)
		}
	}()

	ctx := context.Background()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	g.Log().Infof(ctx, "Received signal: %v, shutting down...", sig)

	a.Stop()
	os.Exit(0)
}
