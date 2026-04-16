// =================================================================================
// Monitor Entity Definitions
// =================================================================================

package entity

import "time"

// ServiceStatus 服务状态
type ServiceStatus struct {
	Status       string    `json:"status"`        // running, stopped, error
	Version      string    `json:"version"`       // 服务版本
	StartTime    time.Time `json:"startTime"`     // 启动时间
	Uptime       int64     `json:"uptime"`        // 运行时长(秒)
	BinlogFile   string    `json:"binlogFile"`    // 当前binlog文件
	BinlogPos    uint32    `json:"binlogPos"`     // 当前binlog位置
	GTID         string    `json:"gtid"`          // GTID
	DelaySeconds int64     `json:"delaySeconds"`  // 同步延迟(秒)
	QPS          float64   `json:"qps"`           // 每秒查询数
	TPS          float64   `json:"tps"`           // 每秒事务数
}

// TargetStatus 目标状态
type TargetStatus struct {
	Name         string    `json:"name"`          // 目标名称
	Type         string    `json:"type"`          // clickhouse, elasticsearch, mysql
	Status       string    `json:"status"`        // connected, disconnected, error
	LastSyncTime time.Time `json:"lastSyncTime"`  // 最后同步时间
	TotalEvents  int64     `json:"totalEvents"`   // 总事件数
	TotalErrors  int64     `json:"totalErrors"`   // 总错误数
	LastError    string    `json:"lastError"`     // 最后错误信息
	IsEnabled    bool      `json:"isEnabled"`     // 是否启用
}

// SyncEvent 同步事件
type SyncEvent struct {
	Timestamp   time.Time `json:"timestamp"`   // 事件时间
	EventType   string    `json:"eventType"`   // INSERT, UPDATE, DELETE
	TargetName  string    `json:"targetName"`  // 目标名称
	TableName   string    `json:"tableName"`   // 表名
	RowsCount   int       `json:"rowsCount"`   // 影响行数
	BinlogFile  string    `json:"binlogFile"`  // binlog文件
	BinlogPos   uint32    `json:"binlogPos"`   // binlog位置
	DurationMs  int       `json:"durationMs"`  // 处理耗时(ms)
	Success     bool      `json:"success"`     // 是否成功
	ErrorMsg    string    `json:"errorMsg"`    // 错误信息
}

// SyncError 同步错误
type SyncError struct {
	Timestamp  time.Time `json:"timestamp"`  // 错误时间
	Level      string    `json:"level"`      // ERROR, WARNING
	TargetName string    `json:"targetName"` // 目标名称
	TableName  string    `json:"tableName"`  // 表名
	Message    string    `json:"message"`    // 错误消息
	StackTrace string    `json:"stackTrace"` // 堆栈信息
	RawData    string    `json:"rawData"`    // 原始数据JSON
	RetryCount int       `json:"retryCount"` // 重试次数
}

// SyncPosition 同步位置
type SyncPosition struct {
	Timestamp    time.Time `json:"timestamp"`    // 记录时间
	TargetName   string    `json:"targetName"`   // 目标名称
	BinlogFile   string    `json:"binlogFile"`   // binlog文件
	BinlogPos    uint32    `json:"binlogPos"`    // binlog位置
	GTID         string    `json:"gtid"`         // GTID
	DelaySeconds int64     `json:"delaySeconds"` // 延迟秒数
}

// SyncMetric 同步指标
type SyncMetric struct {
	Timestamp   time.Time         `json:"timestamp"`   // 指标时间
	MetricName  string            `json:"metricName"`  // 指标名称
	MetricValue float64           `json:"metricValue"` // 指标值
	TargetName  string            `json:"targetName"`  // 目标名称
	TableName   string            `json:"tableName"`   // 表名
	Tags        map[string]string `json:"tags"`        // 额外标签
}

// EventStats 事件统计
type EventStats struct {
	TotalInsert int64 `json:"totalInsert"` // 总INSERT数
	TotalUpdate int64 `json:"totalUpdate"` // 总UPDATE数
	TotalDelete int64 `json:"totalDelete"` // 总DELETE数
	TotalRows   int64 `json:"totalRows"`   // 总行数
	TotalErrors int64 `json:"totalErrors"` // 总错误数
	AvgDuration int64 `json:"avgDuration"` // 平均耗时(ms)
	MaxDuration int64 `json:"maxDuration"` // 最大耗时(ms)
	MinDuration int64 `json:"minDuration"` // 最小耗时(ms)
}

// LatencyStats 延迟统计
type LatencyStats struct {
	CurrentDelay int64   `json:"currentDelay"` // 当前延迟(秒)
	MaxDelay     int64   `json:"maxDelay"`     // 最大延迟(秒)
	MinDelay     int64   `json:"minDelay"`     // 最小延迟(秒)
	AvgDelay     float64 `json:"avgDelay"`     // 平均延迟(秒)
	P95Delay     int64   `json:"p95Delay"`     // P95延迟(秒)
	P99Delay     int64   `json:"p99Delay"`     // P99延迟(秒)
}

// ErrorStats 错误统计
type ErrorStats struct {
	TotalErrors   int64            `json:"totalErrors"`   // 总错误数
	TotalWarnings int64            `json:"totalWarnings"` // 总警告数
	ByLevel       map[string]int64 `json:"byLevel"`       // 按级别统计
	ByTarget      map[string]int64 `json:"byTarget"`      // 按目标统计
	ByTable       map[string]int64 `json:"byTable"`       // 按表统计
}

// PositionHistory 位置历史
type PositionHistory struct {
	Timestamp    time.Time `json:"timestamp"`    // 时间点
	BinlogFile   string    `json:"binlogFile"`   // binlog文件
	BinlogPos    uint32    `json:"binlogPos"`    // binlog位置
	DelaySeconds int64     `json:"delaySeconds"` // 延迟秒数
}

// MetricHistory 指标历史
type MetricHistory struct {
	Timestamp   time.Time `json:"timestamp"`   // 时间点
	MetricName  string    `json:"metricName"`  // 指标名称
	MetricValue float64   `json:"metricValue"` // 指标值
}

// HealthCheck 健康检查结果
type HealthCheck struct {
	Status      string            `json:"status"`      // healthy, unhealthy, degraded
	Checks      map[string]Check  `json:"checks"`      // 各项检查结果
	LastChecked time.Time         `json:"lastChecked"` // 最后检查时间
}

// Check 单项检查结果
type Check struct {
	Status  string `json:"status"`  // ok, error, warning
	Message string `json:"message"` // 检查消息
	Latency int64  `json:"latency"` // 响应延迟(ms)
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Enabled       bool `yaml:"enabled" json:"enabled"`             // 是否启用监控
	HistoryDays   int  `yaml:"historyDays" json:"historyDays"`     // 历史数据保留天数
	CollectPeriod int  `yaml:"collectPeriod" json:"collectPeriod"` // 采集周期(秒)
	MaxEvents     int  `yaml:"maxEvents" json:"maxEvents"`         // 内存最大事件数
	MaxErrors     int  `yaml:"maxErrors" json:"maxErrors"`         // 内存最大错误数
}

// SetDefaults 设置默认值（仅对零值设置）
func (c *MonitorConfig) SetDefaults() {
	if c.HistoryDays == 0 {
		c.HistoryDays = 30
	}
	if c.CollectPeriod == 0 {
		c.CollectPeriod = 10
	}
	if c.MaxEvents == 0 {
		c.MaxEvents = 10000
	}
	if c.MaxErrors == 0 {
		c.MaxErrors = 1000
	}
}
