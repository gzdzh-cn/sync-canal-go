// =================================================================================
// Monitor Config - 监控配置加载
// =================================================================================

package monitor

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"sync-canal-go/internal/model/entity"
)

// LoadConfig 加载监控配置
func LoadConfig(ctx context.Context) (*entity.MonitorConfig, *entity.ClickHouseConfig) {
	cfg := &entity.MonitorConfig{
		Enabled:       g.Cfg().MustGet(ctx, "monitor.enabled", true).Bool(),
		HistoryDays:   g.Cfg().MustGet(ctx, "monitor.historyDays", 30).Int(),
		CollectPeriod: g.Cfg().MustGet(ctx, "monitor.collectPeriod", 10).Int(),
		MaxEvents:     g.Cfg().MustGet(ctx, "monitor.maxEvents", 10000).Int(),
		MaxErrors:     g.Cfg().MustGet(ctx, "monitor.maxErrors", 1000).Int(),
	}

	chCfg := &entity.ClickHouseConfig{
		Host:     g.Cfg().MustGet(ctx, "monitor.clickhouse.host", "127.0.0.1").String(),
		Port:     g.Cfg().MustGet(ctx, "monitor.clickhouse.port", 8124).Int(),
		User:     g.Cfg().MustGet(ctx, "monitor.clickhouse.user", "default").String(),
		Password: g.Cfg().MustGet(ctx, "monitor.clickhouse.password", "").String(),
		Database: g.Cfg().MustGet(ctx, "monitor.clickhouse.database", "sync_monitor").String(),
	}

	return cfg, chCfg
}
