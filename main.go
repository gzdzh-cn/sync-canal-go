package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"gopkg.in/yaml.v3"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/go-mysql-org/go-mysql/schema"
)

// DailyLogger 按日期分割的日志记录器
type DailyLogger struct {
	logDir     string
	currentDay string
	logFile    *os.File
	logger     *log.Logger
	mu         sync.Mutex
}

// NewDailyLogger 创建按日期分割的日志记录器
func NewDailyLogger(logDir string) (*DailyLogger, error) {
	// 创建日志目录
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory failed: %v", err)
	}

	dl := &DailyLogger{
		logDir: logDir,
		logger: log.New(io.Discard, "", log.LstdFlags),
	}

	// 初始化日志文件
	if err := dl.rotateLogFile(); err != nil {
		return nil, err
	}

	return dl, nil
}

// rotateLogFile 轮转日志文件（按日期）
func (dl *DailyLogger) rotateLogFile() error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// 获取当前日期
	today := time.Now().Format("2006-01-02")

	// 如果日期没变，不需要轮转
	if today == dl.currentDay && dl.logFile != nil {
		return nil
	}

	// 关闭旧文件
	if dl.logFile != nil {
		dl.logFile.Close()
	}

	// 创建新文件
	logPath := filepath.Join(dl.logDir, today+".log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file failed: %v", err)
	}

	dl.logFile = file
	dl.currentDay = today
	dl.logger.SetOutput(file)

	return nil
}

// Write 实现 io.Writer 接口
func (dl *DailyLogger) Write(p []byte) (n int, err error) {
	// 检查是否需要轮转
	if time.Now().Format("2006-01-02") != dl.currentDay {
		dl.rotateLogFile()
	}

	dl.mu.Lock()
	defer dl.mu.Unlock()

	if dl.logFile == nil {
		return 0, fmt.Errorf("log file not opened")
	}

	return dl.logFile.Write(p)
}

// Close 关闭日志记录器
func (dl *DailyLogger) Close() error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	if dl.logFile != nil {
		return dl.logFile.Close()
	}
	return nil
}

// MultiWriter 同时写入多个目标
type MultiWriter struct {
	writers []io.Writer
}

// Write 实现 io.Writer 接口
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
	}
	return len(p), nil
}

// 全局日志记录器
var dailyLogger *DailyLogger

// Config 配置结构
type Config struct {
	MySQL      MySQLConfig      `yaml:"mysql"`
	ClickHouse ClickHouseConfig `yaml:"clickhouse"`
	Sync       SyncConfig       `yaml:"sync"`
	Log        LogConfig        `yaml:"log"`
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	ServerID uint32 `yaml:"server_id"`
	Flavor   string `yaml:"flavor"`
}

// ClickHouseConfig ClickHouse 配置
type ClickHouseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// SyncConfig 同步配置
type SyncConfig struct {
	Database        string   `yaml:"database"`
	Tables          []string `yaml:"tables"`
	FullSync        bool     `yaml:"full_sync"`
	BatchSize       int      `yaml:"batch_size"`
	OptimizeInterval int     `yaml:"optimize_interval"` // OPTIMIZE 清理间隔（分钟），0=禁用，默认1440（24小时）
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level"`
	File   string `yaml:"file"`
	Stdout bool   `yaml:"stdout"`
}

// ClickHouseHandler ClickHouse 事件处理器
type ClickHouseHandler struct {
	canal.DummyEventHandler
	chConn  driver.Conn
	config  *Config
	batch   driver.Batch
	batchMu sync.Mutex
}

// NewClickHouseHandler 创建处理器
func NewClickHouseHandler(chConn driver.Conn, config *Config) *ClickHouseHandler {
	return &ClickHouseHandler{
		chConn: chConn,
		config: config,
	}
}

// OnRow 处理行变更事件
func (h *ClickHouseHandler) OnRow(e *canal.RowsEvent) error {
	// 只处理配置中指定的表
	tableName := e.Table.Name
	allowed := false
	for _, t := range h.config.Sync.Tables {
		if t == tableName {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil // 忽略非目标表
	}

	switch e.Action {
	case canal.InsertAction:
		return h.handleInsert(e)
	case canal.UpdateAction:
		return h.handleUpdate(e)
	case canal.DeleteAction:
		return h.handleDelete(e)
	}
	return nil
}

// handleInsert 处理 INSERT 事件
func (h *ClickHouseHandler) handleInsert(e *canal.RowsEvent) error {
	ctx := context.Background()

	// 构建 INSERT SQL
	sql := fmt.Sprintf("INSERT INTO %s.%s", h.config.ClickHouse.Database, e.Table.Name)

	// 批量插入
	batch, err := h.chConn.PrepareBatch(ctx, sql)
	if err != nil {
		log.Printf("[ERROR] Prepare batch failed: %v", err)
		return err
	}

	// 插入所有行
	for _, row := range e.Rows {
		values := h.convertRow(row, e.Table.Columns)
		if err := batch.Append(values...); err != nil {
			log.Printf("[ERROR] Append row failed: %v", err)
			continue
		}
	}

	// 发送批量数据
	if err := batch.Send(); err != nil {
		log.Printf("[ERROR] Send batch failed: %v", err)
		return err
	}

	log.Printf("[INFO] Inserted %d rows to %s.%s", len(e.Rows), h.config.ClickHouse.Database, e.Table.Name)
	return nil
}

// handleUpdate 处理 UPDATE 事件
func (h *ClickHouseHandler) handleUpdate(e *canal.RowsEvent) error {
	ctx := context.Background()

	// ClickHouse 使用 ALTER TABLE UPDATE
	// 注意: ClickHouse 的 UPDATE 是异步的，性能较差
	// 建议: 使用 ReplacingMergeTree 引擎，通过版本号处理更新

	for i := 0; i < len(e.Rows); i += 2 {
		// e.Rows[i] 是旧值, e.Rows[i+1] 是新值
		newRow := e.Rows[i+1]

		// 简化处理: 直接插入新版本
		// 实际项目中建议使用 ReplacingMergeTree
		sql := fmt.Sprintf("INSERT INTO %s.%s", h.config.ClickHouse.Database, e.Table.Name)
		batch, err := h.chConn.PrepareBatch(ctx, sql)
		if err != nil {
			log.Printf("[ERROR] Prepare batch failed: %v", err)
			continue
		}

		values := h.convertRow(newRow, e.Table.Columns)
		if err := batch.Append(values...); err != nil {
			log.Printf("[ERROR] Append row failed: %v", err)
			continue
		}

		if err := batch.Send(); err != nil {
			log.Printf("[ERROR] Send batch failed: %v", err)
			continue
		}
	}

	log.Printf("[INFO] Updated %d rows in %s.%s", len(e.Rows)/2, h.config.ClickHouse.Database, e.Table.Name)
	return nil
}

// handleDelete 处理 DELETE 事件
func (h *ClickHouseHandler) handleDelete(e *canal.RowsEvent) error {
	ctx := context.Background()

	// ClickHouse 使用 ALTER TABLE DELETE
	// 注意: DELETE 是异步的，性能较差
	// 建议: 使用 CollapsingMergeTree 或 TTL 处理删除

	pkIndex := h.findPrimaryKeyIndex(e.Table)

	for _, row := range e.Rows {
		sql := fmt.Sprintf(
			"ALTER TABLE %s.%s DELETE WHERE id = ?",
			h.config.ClickHouse.Database,
			e.Table.Name,
		)

		if err := h.chConn.Exec(ctx, sql, row[pkIndex]); err != nil {
			log.Printf("[ERROR] Delete failed: %v", err)
			continue
		}
	}

	log.Printf("[INFO] Deleted %d rows from %s.%s", len(e.Rows), h.config.ClickHouse.Database, e.Table.Name)
	return nil
}

// convertRow 转换行数据 (为 ReplacingMergeTree 添加 _version 列)
func (h *ClickHouseHandler) convertRow(row []interface{}, columns []schema.TableColumn) []interface{} {
	values := make([]interface{}, len(row))
	for i, v := range row {
		values[i] = v
	}
	// 添加 _version 列 (使用当前时间戳作为版本号)
	values = append(values, time.Now().UnixNano())
	return values
}

// findPrimaryKeyIndex 查找主键索引
func (h *ClickHouseHandler) findPrimaryKeyIndex(table *schema.Table) int {
	// 查找主键列
	for i, col := range table.Columns {
		// 简化判断: 假设主键名为 id
		if col.Name == "id" {
			return i
		}
	}
	return 0
}

// OnRotate 处理 binlog 轮转
func (h *ClickHouseHandler) OnRotate(header *replication.EventHeader, e *replication.RotateEvent) error {
	log.Printf("[INFO] Rotate to %s:%d", string(e.NextLogName), e.Position)
	return nil
}

// OnDDL 处理 DDL 事件
func (h *ClickHouseHandler) OnDDL(header *replication.EventHeader, nextPos mysql.Position, queryEvent *replication.QueryEvent) error {
	log.Printf("[INFO] DDL: %s", string(queryEvent.Query))
	return nil
}

// optimizeTable 执行表优化，清理旧版本数据
func (h *ClickHouseHandler) optimizeTable() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	for _, table := range h.config.Sync.Tables {
		startTime := time.Now()
		
		log.Printf("[INFO] Starting daily OPTIMIZE for table %s.%s", h.config.ClickHouse.Database, table)
		
		// 执行 OPTIMIZE
		sql := fmt.Sprintf("OPTIMIZE TABLE %s.%s FINAL", h.config.ClickHouse.Database, table)
		if err := h.chConn.Exec(ctx, sql); err != nil {
			log.Printf("[WARN] Optimize table %s failed: %v", table, err)
			continue
		}
		
		elapsed := time.Since(startTime)
		log.Printf("[INFO] OPTIMIZE completed for %s.%s in %v", 
			h.config.ClickHouse.Database, table, elapsed)
	}
}

// startPeriodicOptimize 启动定时清理任务
func (h *ClickHouseHandler) startPeriodicOptimize() {
	// 获取配置的间隔时间（分钟）
	intervalMinutes := h.config.Sync.OptimizeInterval
	if intervalMinutes == 0 {
		log.Printf("[INFO] Periodic OPTIMIZE is disabled (optimize_interval=0)")
		return
	}
	
	// 默认 24 小时
	if intervalMinutes < 0 {
		intervalMinutes = 1440
	}
	
	interval := time.Duration(intervalMinutes) * time.Minute

	go func() {
		for {
			log.Printf("[INFO] Next OPTIMIZE scheduled in %v", interval)
			
			// 等待到执行时间
			timer := time.NewTimer(interval)
			<-timer.C
			
			// 执行优化
			h.optimizeTable()
		}
	}()
}

// String 返回处理器名称
func (h *ClickHouseHandler) String() string {
	return "ClickHouseHandler"
}

// loadConfig 加载配置文件
func loadConfig(configFile string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("read config file failed: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config file failed: %v", err)
	}

	// 设置默认值
	if config.MySQL.Flavor == "" {
		config.MySQL.Flavor = "mysql"
	}
	if config.Sync.BatchSize == 0 {
		config.Sync.BatchSize = 1000
	}

	return &config, nil
}

// connectClickHouse 连接 ClickHouse
func connectClickHouse(config *ClickHouseConfig) (driver.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.User,
			Password: config.Password,
		},
		Protocol: clickhouse.HTTP, // 使用 HTTP 协议（端口 8123），Native 端口 9000 被其他服务占用
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: time.Second * 30,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("connect to clickhouse failed: %v", err)
	}

	// 测试连接
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping clickhouse failed: %v", err)
	}

	return conn, nil
}

// createCanal 创建 Canal
func createCanal(config *Config) (*canal.Canal, error) {
	cfg := canal.NewDefaultConfig()
	cfg.Addr = fmt.Sprintf("%s:%d", config.MySQL.Host, config.MySQL.Port)
	cfg.User = config.MySQL.User
	cfg.Password = config.MySQL.Password
	cfg.Flavor = config.MySQL.Flavor
	cfg.ServerID = config.MySQL.ServerID

	// 只监听指定的数据库和表
	if len(config.Sync.Tables) > 0 {
		for _, table := range config.Sync.Tables {
			cfg.IncludeTableRegex = append(cfg.IncludeTableRegex,
				fmt.Sprintf("%s\\.%s", config.Sync.Database, table))
		}
	}

	c, err := canal.NewCanal(cfg)
	if err != nil {
		return nil, fmt.Errorf("create canal failed: %v", err)
	}

	return c, nil
}

func main() {
	// 解析命令行参数
	configFile := flag.String("config", "", "配置文件路径 (默认: config/config.yaml)")
	flag.Parse()

	// 设置默认配置文件路径
	configPath := *configFile
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	// 加载配置
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("[FATAL] Load config failed: %v", err)
	}

	// 初始化日志记录器
	dailyLogger, err = NewDailyLogger("./logs")
	if err != nil {
		log.Fatalf("[FATAL] Init logger failed: %v", err)
	}
	defer dailyLogger.Close()

	// 设置日志输出（同时输出到文件和标准输出）
	if config.Log.Stdout {
		log.SetOutput(&MultiWriter{
			writers: []io.Writer{dailyLogger, os.Stdout},
		})
	} else {
		log.SetOutput(dailyLogger)
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Printf("[INFO] Config loaded: %+v", config)

	// 连接 ClickHouse
	chConn, err := connectClickHouse(&config.ClickHouse)
	if err != nil {
		log.Fatalf("[FATAL] Connect ClickHouse failed: %v", err)
	}
	defer chConn.Close()

	log.Printf("[INFO] Connected to ClickHouse: %s:%d", config.ClickHouse.Host, config.ClickHouse.Port)

	// 创建 Canal
	c, err := createCanal(config)
	if err != nil {
		log.Fatalf("[FATAL] Create canal failed: %v", err)
	}
	defer c.Close()

	log.Printf("[INFO] Canal created, listening MySQL: %s:%d", config.MySQL.Host, config.MySQL.Port)

	// 设置事件处理器
	handler := NewClickHouseHandler(chConn, config)
	c.SetEventHandler(handler)

	// 启动定时清理任务
	handler.startPeriodicOptimize()

	// 获取当前 binlog 位置
	pos, err := c.GetMasterPos()
	if err != nil {
		log.Fatalf("[FATAL] Get master position failed: %v", err)
	}

	log.Printf("[INFO] Start syncing from %s:%d", pos.Name, pos.Pos)

	// 信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动同步
	go func() {
		if err := c.RunFrom(pos); err != nil {
			log.Fatalf("[FATAL] Run canal failed: %v", err)
		}
	}()

	// 等待退出信号
	sig := <-sigChan
	log.Printf("[INFO] Received signal: %v, shutting down...", sig)

	// 保存断点
	currentPos, err := c.GetMasterPos()
	if err == nil {
		log.Printf("[INFO] Current position: %s:%d", currentPos.Name, currentPos.Pos)
	}

	log.Println("[INFO] Sync service stopped")
}
