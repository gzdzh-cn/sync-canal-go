// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

// CanalConfig Canal 配置
type CanalConfig struct {
	ServerId uint32 `yaml:"serverId" json:"serverId"`
	Flavor   string `yaml:"flavor" json:"flavor"`
	Dump     bool   `yaml:"dump" json:"dump"`
}

// ScheduleConfig 定时任务配置
type ScheduleConfig struct {
	Enable   bool `yaml:"enable" json:"enable"`
	Interval int  `yaml:"interval" json:"interval"` // 分钟
}

// ClickHouseConfig ClickHouse 配置
type ClickHouseConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	Database string `yaml:"database" json:"database"`
}

// ElasticsearchConfig Elasticsearch 配置
type ElasticsearchConfig struct {
	Hosts    []string `yaml:"hosts" json:"hosts"`
	Username string   `yaml:"username" json:"username"`
	Password string   `yaml:"password" json:"password"`
	Index    string   `yaml:"index" json:"index"` // 索引名前缀，留空则使用表名
}

// MySQLTargetConfig MySQL 目标配置
type MySQLTargetConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	Database string `yaml:"database" json:"database"`
}

// TargetConfig 同步目标配置
type TargetConfig struct {
	Name          string              `yaml:"name" json:"name"`
	Type          string              `yaml:"type" json:"type"`                     // clickhouse, elasticsearch, mysql
	Tables        []string            `yaml:"tables" json:"tables"`                 // 同步的表列表
	ClickHouse    *ClickHouseConfig   `yaml:"clickhouse" json:"clickhouse"`         // ClickHouse 配置
	Elasticsearch *ElasticsearchConfig `yaml:"elasticsearch" json:"elasticsearch"`   // ES 配置
	MySQL         *MySQLTargetConfig  `yaml:"mysql" json:"mysql"`                   // MySQL 目标配置
	Schedule      *ScheduleConfig     `yaml:"schedule" json:"schedule"`             // 定时任务配置
}

// SyncConfig 同步配置
type SyncConfig struct {
	Database  string         `yaml:"database" json:"database"`   // 源数据库
	BatchSize int            `yaml:"batchSize" json:"batchSize"` // 批量大小
	Targets   []TargetConfig `yaml:"targets" json:"targets"`     // 同步目标列表
}
