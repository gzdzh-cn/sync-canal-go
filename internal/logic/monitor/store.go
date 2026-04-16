// =================================================================================
// Metrics Store - 指标存储
// =================================================================================

package monitor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gogf/gf/v2/frame/g"

	"sync-canal-go/internal/model/entity"
	"sync-canal-go/internal/service"
)

// Store 指标存储实现
type Store struct {
	chConn driver.Conn
	config *entity.MonitorConfig

	// 内存缓存
	eventCache  map[string]*entity.SyncEvent
	errorCache  map[string]*entity.SyncError
	positionMap map[string]*entity.SyncPosition
	mu          sync.RWMutex

	// 批量写入缓冲
	eventBuffer  []*entity.SyncEvent
	metricBuffer []*entity.SyncMetric
	bufferMu     sync.Mutex
}

// NewStore 创建存储实例
func NewStore(config *entity.MonitorConfig, chConfig *entity.ClickHouseConfig) (*Store, error) {
	s := &Store{
		config:      config,
		eventCache:  make(map[string]*entity.SyncEvent),
		errorCache:  make(map[string]*entity.SyncError),
		positionMap: make(map[string]*entity.SyncPosition),
	}

	// 连接 ClickHouse (可选，失败则使用内存模式)
	if chConfig != nil && chConfig.Host != "" {
		// 判断端口类型：8123/8124 是 HTTP 端口，9000 是原生 TCP 端口
		var conn driver.Conn
		var err error

		if chConfig.Port == 8123 || chConfig.Port == 8124 {
			// 使用 HTTP 协议连接
			g.Log().Infof(context.Background(), "Connecting to ClickHouse via HTTP protocol on port %d", chConfig.Port)
			conn, err = clickhouse.Open(&clickhouse.Options{
				Addr: []string{fmt.Sprintf("%s:%d", chConfig.Host, chConfig.Port)},
				Auth: clickhouse.Auth{
					Database: "default",
					Username: chConfig.User,
					Password: chConfig.Password,
				},
				Settings: clickhouse.Settings{
					"max_execution_time": 60,
				},
				DialTimeout: time.Second * 30,
				Protocol:    clickhouse.HTTP,
			})
		} else {
			// 使用原生 TCP 协议连接（适用于 9000）
			g.Log().Infof(context.Background(), "Connecting to ClickHouse via TCP protocol on port %d", chConfig.Port)
			conn, err = clickhouse.Open(&clickhouse.Options{
				Addr: []string{fmt.Sprintf("%s:%d", chConfig.Host, chConfig.Port)},
				Auth: clickhouse.Auth{
					Database: "default",
					Username: chConfig.User,
					Password: chConfig.Password,
				},
				Settings: clickhouse.Settings{
					"max_execution_time": 60,
				},
				DialTimeout: time.Second * 30,
			})
		}
		if err != nil {
			g.Log().Warningf(context.Background(), "Connect to ClickHouse failed: %v, using memory mode", err)
			return s, nil
		}

		if err := conn.Ping(context.Background()); err != nil {
			g.Log().Warningf(context.Background(), "Ping ClickHouse failed: %v, using memory mode", err)
			return s, nil
		}

		// 创建数据库（如果不存在）
		ctx := context.Background()
		if err := createDatabase(ctx, conn, chConfig.Database); err != nil {
			g.Log().Warningf(ctx, "Create database failed: %v, using memory mode", err)
			return s, nil
		}

		// 切换到目标数据库
		conn.Close()
		if chConfig.Port == 8123 || chConfig.Port == 8124 {
			// 使用 HTTP 协议连接
			conn, err = clickhouse.Open(&clickhouse.Options{
				Addr: []string{fmt.Sprintf("%s:%d", chConfig.Host, chConfig.Port)},
				Auth: clickhouse.Auth{
					Database: chConfig.Database,
					Username: chConfig.User,
					Password: chConfig.Password,
				},
				Settings: clickhouse.Settings{
					"max_execution_time": 60,
				},
				DialTimeout: time.Second * 30,
				Protocol:    clickhouse.HTTP,
			})
		} else {
			// 使用原生 TCP 协议连接
			conn, err = clickhouse.Open(&clickhouse.Options{
				Addr: []string{fmt.Sprintf("%s:%d", chConfig.Host, chConfig.Port)},
				Auth: clickhouse.Auth{
					Database: chConfig.Database,
					Username: chConfig.User,
					Password: chConfig.Password,
				},
				Settings: clickhouse.Settings{
					"max_execution_time": 60,
				},
				DialTimeout: time.Second * 30,
			})
		}
		if err != nil {
			g.Log().Warningf(ctx, "Connect to target database failed: %v, using memory mode", err)
			return s, nil
		}

		// 创建表
		historyDays := 30
		if config != nil && config.HistoryDays > 0 {
			historyDays = config.HistoryDays
		}
		if err := createTables(ctx, conn, historyDays); err != nil {
			g.Log().Warningf(ctx, "Create tables failed: %v, using memory mode", err)
			return s, nil
		}

		s.chConn = conn
		g.Log().Infof(ctx, "Connected to ClickHouse for monitoring: %s:%d/%s", chConfig.Host, chConfig.Port, chConfig.Database)
	}

	return s, nil
}

// createDatabase 创建数据库
func createDatabase(ctx context.Context, conn driver.Conn, database string) error {
	return conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", database))
}

// createTables 创建监控表
func createTables(ctx context.Context, conn driver.Conn, historyDays int) error {
	if historyDays <= 0 {
		historyDays = 30 // 默认 30 天
	}

	// 监控指标表 - 使用 DateTime 而非 DateTime64 以兼容 TTL
	if err := conn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS sync_metrics (
			timestamp DateTime,
			metric_name String,
			metric_value Float64,
			target_name String,
			table_name String,
			tags Map(String, String)
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(timestamp)
		ORDER BY (timestamp, metric_name)
		TTL timestamp + INTERVAL %d DAY
	`, historyDays)); err != nil {
		return fmt.Errorf("create sync_metrics table: %v", err)
	}

	// 同步事件表
	if err := conn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS sync_events (
			timestamp DateTime,
			event_type String,
			target_name String,
			table_name String,
			rows_count UInt32,
			binlog_file String,
			binlog_pos UInt64,
			duration_ms UInt32,
			success UInt8,
			error_msg String
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(timestamp)
		ORDER BY (timestamp, target_name, table_name)
		TTL timestamp + INTERVAL %d DAY
	`, historyDays)); err != nil {
		return fmt.Errorf("create sync_events table: %v", err)
	}

	// 同步位置表
	if err := conn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS sync_positions (
			timestamp DateTime,
			target_name String,
			binlog_file String,
			binlog_pos UInt64,
			gtid String,
			delay_seconds UInt32
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(timestamp)
		ORDER BY (timestamp, target_name)
		TTL timestamp + INTERVAL %d DAY
	`, historyDays)); err != nil {
		return fmt.Errorf("create sync_positions table: %v", err)
	}

	// 错误日志表
	if err := conn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS sync_errors (
			timestamp DateTime,
			level String,
			target_name String,
			table_name String,
			message String,
			stack_trace String,
			raw_data String
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(timestamp)
		ORDER BY (timestamp, level)
		TTL timestamp + INTERVAL %d DAY
	`, historyDays)); err != nil {
		return fmt.Errorf("create sync_errors table: %v", err)
	}

	g.Log().Infof(ctx, "Monitor tables created successfully with TTL %d days", historyDays)
	return nil
}

// Close 关闭连接
func (s *Store) Close() error {
	if s.chConn != nil {
		return s.chConn.Close()
	}
	return nil
}

// SaveEvent 保存事件
func (s *Store) SaveEvent(ctx context.Context, e *entity.SyncEvent) error {
	// 内存缓存
	s.mu.Lock()
	s.eventCache[fmt.Sprintf("%d_%s", e.Timestamp.UnixNano(), e.TableName)] = e
	s.mu.Unlock()

	// ClickHouse 写入
	if s.chConn != nil {
		return s.insertEvent(ctx, e)
	}
	return nil
}

// SaveEvents 批量保存事件
func (s *Store) SaveEvents(ctx context.Context, events []*entity.SyncEvent) error {
	if s.chConn == nil || len(events) == 0 {
		return nil
	}

	batch, err := s.chConn.PrepareBatch(ctx, "INSERT INTO sync_events")
	if err != nil {
		return fmt.Errorf("prepare batch failed: %v", err)
	}

	for _, e := range events {
		if err := batch.Append(
			e.Timestamp,
			e.EventType,
			e.TargetName,
			e.TableName,
			e.RowsCount,
			e.BinlogFile,
			e.BinlogPos,
			e.DurationMs,
			boolToInt(e.Success),
			e.ErrorMsg,
		); err != nil {
			g.Log().Warningf(ctx, "Append event failed: %v", err)
		}
	}

	return batch.Send()
}

// insertEvent 插入单个事件
func (s *Store) insertEvent(ctx context.Context, e *entity.SyncEvent) error {
	return s.chConn.Exec(ctx, `
		INSERT INTO sync_events (
			timestamp, event_type, target_name, table_name,
			rows_count, binlog_file, binlog_pos, duration_ms, success, error_msg
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Timestamp, e.EventType, e.TargetName, e.TableName,
		e.RowsCount, e.BinlogFile, e.BinlogPos, e.DurationMs,
		boolToInt(e.Success), e.ErrorMsg,
	)
}

// GetEvents 查询事件列表
func (s *Store) GetEvents(ctx context.Context, query *service.EventQuery) ([]*entity.SyncEvent, int64, error) {
	if s.chConn == nil {
		return s.getEventsFromMemory(query)
	}

	// 构建查询
	sql := "SELECT timestamp, event_type, target_name, table_name, rows_count, binlog_file, binlog_pos, duration_ms, success, error_msg FROM sync_events WHERE 1=1"
	args := []any{}

	if query.TargetName != "" {
		sql += " AND target_name = ?"
		args = append(args, query.TargetName)
	}
	if query.TableName != "" {
		sql += " AND table_name = ?"
		args = append(args, query.TableName)
	}
	if query.EventType != "" {
		sql += " AND lower(event_type) = lower(?)"
		args = append(args, query.EventType)
	}
	if query.StartTime > 0 {
		sql += " AND timestamp >= ?"
		args = append(args, time.Unix(query.StartTime, 0))
	}
	if query.EndTime > 0 {
		sql += " AND timestamp <= ?"
		args = append(args, time.Unix(query.EndTime, 0))
	}

	// 计算总数
	countSQL := "SELECT count() FROM (" + sql + ")"
	var total uint64
	if err := s.chConn.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 分页
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	offset := (query.Page - 1) * query.PageSize
	sql += " ORDER BY timestamp DESC LIMIT ?, ?"
	args = append(args, offset, query.PageSize)

	// 查询数据
	rows, err := s.chConn.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	events := make([]*entity.SyncEvent, 0)
	for rows.Next() {
		e := &entity.SyncEvent{}
		var success uint8
		var rowsCount, durationMs uint32
		var binlogPos uint64
		if err := rows.Scan(
			&e.Timestamp, &e.EventType, &e.TargetName, &e.TableName,
			&rowsCount, &e.BinlogFile, &binlogPos, &durationMs, &success, &e.ErrorMsg,
		); err != nil {
			return nil, 0, err
		}
		e.RowsCount = int(rowsCount)
		e.BinlogPos = uint32(binlogPos)
		e.DurationMs = int(durationMs)
		e.Success = success == 1
		events = append(events, e)
	}

	return events, int64(total), nil
}

// getEventsFromMemory 从内存获取事件
func (s *Store) getEventsFromMemory(query *service.EventQuery) ([]*entity.SyncEvent, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]*entity.SyncEvent, 0)
	for _, e := range s.eventCache {
		// 应用过滤条件
		if query.TargetName != "" && e.TargetName != query.TargetName {
			continue
		}
		if query.TableName != "" && e.TableName != query.TableName {
			continue
		}
		// 兼容大小写
		if query.EventType != "" && !strings.EqualFold(e.EventType, query.EventType) {
			continue
		}
		events = append(events, e)
	}

	total := int64(len(events))

	// 分页
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	start := (query.Page - 1) * query.PageSize
	end := start + query.PageSize
	if start >= len(events) {
		return nil, total, nil
	}
	if end > len(events) {
		end = len(events)
	}

	return events[start:end], total, nil
}

// SaveError 保存错误
func (s *Store) SaveError(ctx context.Context, err *entity.SyncError) error {
	// 内存缓存
	s.mu.Lock()
	s.errorCache[fmt.Sprintf("%d_%s", err.Timestamp.UnixNano(), err.TargetName)] = err
	s.mu.Unlock()

	// ClickHouse 写入
	if s.chConn != nil {
		return s.chConn.Exec(ctx, `
			INSERT INTO sync_errors (
				timestamp, level, target_name, table_name, message, stack_trace, raw_data
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			err.Timestamp, err.Level, err.TargetName, err.TableName,
			err.Message, err.StackTrace, err.RawData,
		)
	}
	return nil
}

// GetErrors 查询错误列表
func (s *Store) GetErrors(ctx context.Context, query *service.ErrorQuery) ([]*entity.SyncError, int64, error) {
	if s.chConn == nil {
		return s.getErrorsFromMemory(query)
	}

	sql := "SELECT timestamp, level, target_name, table_name, message, stack_trace, raw_data FROM sync_errors WHERE 1=1"
	args := []any{}

	if query.Level != "" {
		sql += " AND level = ?"
		args = append(args, query.Level)
	}
	if query.TargetName != "" {
		sql += " AND target_name = ?"
		args = append(args, query.TargetName)
	}
	if query.StartTime > 0 {
		sql += " AND timestamp >= ?"
		args = append(args, time.Unix(query.StartTime, 0))
	}
	if query.EndTime > 0 {
		sql += " AND timestamp <= ?"
		args = append(args, time.Unix(query.EndTime, 0))
	}

	// 计算总数
	countSQL := "SELECT count() FROM (" + sql + ")"
	var total uint64
	if err := s.chConn.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 分页
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	offset := (query.Page - 1) * query.PageSize
	sql += " ORDER BY timestamp DESC LIMIT ?, ?"
	args = append(args, offset, query.PageSize)

	rows, err := s.chConn.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	errors := make([]*entity.SyncError, 0)
	for rows.Next() {
		e := &entity.SyncError{}
		if err := rows.Scan(
			&e.Timestamp, &e.Level, &e.TargetName, &e.TableName,
			&e.Message, &e.StackTrace, &e.RawData,
		); err != nil {
			return nil, 0, err
		}
		errors = append(errors, e)
	}

	return errors, int64(total), nil
}

// getErrorsFromMemory 从内存获取错误
func (s *Store) getErrorsFromMemory(query *service.ErrorQuery) ([]*entity.SyncError, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	errors := make([]*entity.SyncError, 0)
	for _, e := range s.errorCache {
		if query.Level != "" && e.Level != query.Level {
			continue
		}
		if query.TargetName != "" && e.TargetName != query.TargetName {
			continue
		}
		errors = append(errors, e)
	}

	total := int64(len(errors))

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	start := (query.Page - 1) * query.PageSize
	end := start + query.PageSize
	if start >= len(errors) {
		return nil, total, nil
	}
	if end > len(errors) {
		end = len(errors)
	}

	return errors[start:end], total, nil
}

// SavePosition 保存位置
func (s *Store) SavePosition(ctx context.Context, p *entity.SyncPosition) error {
	// 内存缓存
	s.mu.Lock()
	s.positionMap[p.TargetName] = p
	s.mu.Unlock()

	// ClickHouse 写入
	if s.chConn != nil {
		return s.chConn.Exec(ctx, `
			INSERT INTO sync_positions (
				timestamp, target_name, binlog_file, binlog_pos, gtid, delay_seconds
			) VALUES (?, ?, ?, ?, ?, ?)`,
			p.Timestamp, p.TargetName, p.BinlogFile, p.BinlogPos, p.GTID, p.DelaySeconds,
		)
	}
	return nil
}

// GetPosition 获取当前位置
func (s *Store) GetPosition(ctx context.Context, targetName string) (*entity.SyncPosition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if p, ok := s.positionMap[targetName]; ok {
		return p, nil
	}
	return nil, nil
}

// GetPositionHistory 获取位置历史
func (s *Store) GetPositionHistory(ctx context.Context, targetName string, start, end int64) ([]*entity.PositionHistory, error) {
	if s.chConn == nil {
		return nil, nil
	}

	rows, err := s.chConn.Query(ctx, `
		SELECT timestamp, binlog_file, binlog_pos, delay_seconds
		FROM sync_positions
		WHERE target_name = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`, targetName, time.Unix(start, 0), time.Unix(end, 0))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]*entity.PositionHistory, 0)
	for rows.Next() {
		h := &entity.PositionHistory{}
		var binlogPos uint64
		var delaySeconds uint32
		if err := rows.Scan(&h.Timestamp, &h.BinlogFile, &binlogPos, &delaySeconds); err != nil {
			return nil, err
		}
		h.BinlogPos = uint32(binlogPos)
		h.DelaySeconds = int64(delaySeconds)
		history = append(history, h)
	}

	return history, nil
}

// SaveMetric 保存指标
func (s *Store) SaveMetric(ctx context.Context, m *entity.SyncMetric) error {
	if s.chConn == nil {
		return nil
	}

	return s.chConn.Exec(ctx, `
		INSERT INTO sync_metrics (
			timestamp, metric_name, metric_value, target_name, table_name, tags
		) VALUES (?, ?, ?, ?, ?, ?)`,
		m.Timestamp, m.MetricName, m.MetricValue, m.TargetName, m.TableName, m.Tags,
	)
}

// SaveMetrics 批量保存指标
func (s *Store) SaveMetrics(ctx context.Context, metrics []*entity.SyncMetric) error {
	if s.chConn == nil || len(metrics) == 0 {
		return nil
	}

	batch, err := s.chConn.PrepareBatch(ctx, "INSERT INTO sync_metrics")
	if err != nil {
		return err
	}

	for _, m := range metrics {
		if err := batch.Append(
			m.Timestamp, m.MetricName, m.MetricValue, m.TargetName, m.TableName, m.Tags,
		); err != nil {
			g.Log().Warningf(ctx, "Append metric failed: %v", err)
		}
	}

	return batch.Send()
}

// GetMetricHistory 获取指标历史
func (s *Store) GetMetricHistory(ctx context.Context, name string, start, end int64) ([]*entity.MetricHistory, error) {
	if s.chConn == nil {
		return nil, nil
	}

	rows, err := s.chConn.Query(ctx, `
		SELECT timestamp, metric_name, metric_value
		FROM sync_metrics
		WHERE metric_name = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`, name, time.Unix(start, 0), time.Unix(end, 0))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]*entity.MetricHistory, 0)
	for rows.Next() {
		h := &entity.MetricHistory{}
		if err := rows.Scan(&h.Timestamp, &h.MetricName, &h.MetricValue); err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, nil
}

// GetEventStats 获取事件统计
func (s *Store) GetEventStats(ctx context.Context, start, end int64) (*entity.EventStats, error) {
	stats := &entity.EventStats{}

	if s.chConn == nil {
		return stats, nil
	}

	startTime := time.Unix(start, 0)
	endTime := time.Unix(end, 0)

	// 统计各类型事件数量 - 使用 upper() 统一大小写后再分组
	rows, err := s.chConn.Query(ctx, `
		SELECT upper(event_type) as event_type_upper, count() as cnt, sum(rows_count) as rows, avg(duration_ms) as avg_dur, max(duration_ms) as max_dur
		FROM sync_events
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY upper(event_type)
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var cnt, rowsCount uint64
		var avgDur float64
		var maxDur uint32
		if err := rows.Scan(&eventType, &cnt, &rowsCount, &avgDur, &maxDur); err != nil {
			return nil, err
		}

		stats.TotalRows += int64(rowsCount)
		switch eventType {
		case "INSERT":
			stats.TotalInsert = int64(cnt)
		case "UPDATE":
			stats.TotalUpdate = int64(cnt)
		case "DELETE":
			stats.TotalDelete = int64(cnt)
		}
		stats.AvgDuration = int64(avgDur)
		if int64(maxDur) > stats.MaxDuration {
			stats.MaxDuration = int64(maxDur)
		}
	}

	// 统计错误数量
	var errorCount uint64
	err = s.chConn.QueryRow(ctx, `
		SELECT count()
		FROM sync_errors
		WHERE timestamp >= ? AND timestamp <= ?
	`, startTime, endTime).Scan(&errorCount)
	if err == nil {
		stats.TotalErrors = int64(errorCount)
	}

	return stats, nil
}

// GetLatencyStats 获取延迟统计
func (s *Store) GetLatencyStats(ctx context.Context, start, end int64) (*entity.LatencyStats, error) {
	stats := &entity.LatencyStats{}

	if s.chConn == nil {
		g.Log().Warningf(ctx, "GetLatencyStats: chConn is nil")
		return stats, nil
	}

	startTime := time.Unix(start, 0)
	endTime := time.Unix(end, 0)
	g.Log().Debugf(ctx, "GetLatencyStats: querying from %v to %v", startTime, endTime)

	// 获取基本统计 - 使用正确的类型：avg 返回 Float64，max/min 返回 UInt32
	var avgDelay float64
	var maxDelay, minDelay uint32
	var count uint64
	err := s.chConn.QueryRow(ctx, `
		SELECT avg(delay_seconds), max(delay_seconds), min(delay_seconds), count()
		FROM sync_positions
		WHERE timestamp >= ? AND timestamp <= ?
	`, startTime, endTime).Scan(&avgDelay, &maxDelay, &minDelay, &count)

	if err != nil {
		g.Log().Warningf(ctx, "GetLatencyStats query failed: %v", err)
		return stats, nil
	}

	g.Log().Debugf(ctx, "GetLatencyStats: count=%d, avg=%v, max=%v, min=%v", count, avgDelay, maxDelay, minDelay)

	stats.AvgDelay = avgDelay
	stats.MaxDelay = int64(maxDelay)
	stats.MinDelay = int64(minDelay)

	stats.AvgDelay = avgDelay
	stats.MaxDelay = int64(maxDelay)
	stats.MinDelay = int64(minDelay)

	// 获取当前延迟（最新一条记录）
	var currentDelay uint32
	s.chConn.QueryRow(ctx, `
		SELECT delay_seconds
		FROM sync_positions
		WHERE timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp DESC
		LIMIT 1
	`, time.Unix(start, 0), time.Unix(end, 0)).Scan(&currentDelay)
	stats.CurrentDelay = int64(currentDelay)

	// 计算 P95 和 P99
	if count > 0 {
		// P95
		var p95Delay uint32
		p95Offset := int(float64(count) * 0.95)
		if p95Offset < 1 {
			p95Offset = 1
		}
		s.chConn.QueryRow(ctx, `
			SELECT delay_seconds
			FROM sync_positions
			WHERE timestamp >= ? AND timestamp <= ?
			ORDER BY delay_seconds ASC
			LIMIT 1 OFFSET ?
		`, time.Unix(start, 0), time.Unix(end, 0), p95Offset-1).Scan(&p95Delay)
		stats.P95Delay = int64(p95Delay)

		// P99
		var p99Delay uint32
		p99Offset := int(float64(count) * 0.99)
		if p99Offset < 1 {
			p99Offset = 1
		}
		s.chConn.QueryRow(ctx, `
			SELECT delay_seconds
			FROM sync_positions
			WHERE timestamp >= ? AND timestamp <= ?
			ORDER BY delay_seconds ASC
			LIMIT 1 OFFSET ?
		`, time.Unix(start, 0), time.Unix(end, 0), p99Offset-1).Scan(&p99Delay)
		stats.P99Delay = int64(p99Delay)
	}

	return stats, nil
}

// GetLatencyHistory 获取延迟历史
func (s *Store) GetLatencyHistory(ctx context.Context, start, end int64) ([]*entity.MetricHistory, error) {
	if s.chConn == nil {
		return nil, nil
	}

	rows, err := s.chConn.Query(ctx, `
		SELECT timestamp, delay_seconds
		FROM sync_positions
		WHERE timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`, time.Unix(start, 0), time.Unix(end, 0))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]*entity.MetricHistory, 0)
	for rows.Next() {
		var timestamp time.Time
		var delaySeconds uint32
		if err := rows.Scan(&timestamp, &delaySeconds); err != nil {
			return nil, err
		}
		history = append(history, &entity.MetricHistory{
			Timestamp:   timestamp,
			MetricName:  "latency",
			MetricValue: float64(delaySeconds),
		})
	}

	return history, nil
}

// GetErrorStats 获取错误统计
func (s *Store) GetErrorStats(ctx context.Context, start, end int64) (*entity.ErrorStats, error) {
	stats := &entity.ErrorStats{
		ByLevel:  make(map[string]int64),
		ByTarget: make(map[string]int64),
		ByTable:  make(map[string]int64),
	}

	if s.chConn == nil {
		return stats, nil
	}

	// 按级别统计
	rows, err := s.chConn.Query(ctx, `
		SELECT level, count() as cnt
		FROM sync_errors
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY level
	`, time.Unix(start, 0), time.Unix(end, 0))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var level string
		var cnt uint64
		if err := rows.Scan(&level, &cnt); err != nil {
			return nil, err
		}
		stats.ByLevel[level] = int64(cnt)
		if level == "ERROR" {
			stats.TotalErrors = int64(cnt)
		} else if level == "WARNING" {
			stats.TotalWarnings = int64(cnt)
		}
	}

	// 按目标统计
	targetRows, err := s.chConn.Query(ctx, `
		SELECT target_name, count() as cnt
		FROM sync_errors
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY target_name
	`, time.Unix(start, 0), time.Unix(end, 0))
	if err == nil {
		defer targetRows.Close()
		for targetRows.Next() {
			var target string
			var cnt uint64
			if err := targetRows.Scan(&target, &cnt); err == nil {
				stats.ByTarget[target] = int64(cnt)
			}
		}
	}

	// 按表统计
	tableRows, err := s.chConn.Query(ctx, `
		SELECT table_name, count() as cnt
		FROM sync_errors
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY table_name
	`, time.Unix(start, 0), time.Unix(end, 0))
	if err == nil {
		defer tableRows.Close()
		for tableRows.Next() {
			var table string
			var cnt uint64
			if err := tableRows.Scan(&table, &cnt); err == nil {
				stats.ByTable[table] = int64(cnt)
			}
		}
	}

	return stats, nil
}

// GetTargetsFromEvents 从事件数据中获取目标列表
func (s *Store) GetTargetsFromEvents(ctx context.Context) ([]string, error) {
	if s.chConn == nil {
		return nil, nil
	}

	rows, err := s.chConn.Query(ctx, `
		SELECT DISTINCT target_name
		FROM sync_events
		WHERE target_name != ''
		ORDER BY target_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targets := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		targets = append(targets, name)
	}

	return targets, nil
}

// GetTargetStats 获取目标统计信息
func (s *Store) GetTargetStats(ctx context.Context, targetName string, start, end int64) (*entity.TargetStatus, error) {
	status := &entity.TargetStatus{
		Name:      targetName,
		Type:      "clickhouse",
		Status:    "connected",
		IsEnabled: true,
	}

	if s.chConn == nil {
		return status, nil
	}

	// 提取目标名称中的关键部分用于匹配
	// 例如 "ClickHouse:clickhouse-main" -> ["clickhouse-main", "clickhouse-primary"]
	targetNames := s.expandTargetNames(targetName)

	// 构建 IN 查询条件
	inClause := ""
	args := []any{}
	for i, name := range targetNames {
		if i > 0 {
			inClause += ", "
		}
		inClause += "?"
		args = append(args, name)
	}

	// 获取事件统计 - 使用 IN 查询匹配多个可能的名称
	var eventCount uint64
	query := fmt.Sprintf(`
		SELECT count()
		FROM sync_events
		WHERE target_name IN (%s) AND timestamp >= ? AND timestamp <= ?
	`, inClause)
	args = append(args, time.Unix(start, 0), time.Unix(end, 0))
	err := s.chConn.QueryRow(ctx, query, args...).Scan(&eventCount)
	if err != nil {
		eventCount = 0
	}
	status.TotalEvents = int64(eventCount)

	// 获取错误统计
	var errorCount uint64
	query = fmt.Sprintf(`
		SELECT count()
		FROM sync_errors
		WHERE target_name IN (%s) AND timestamp >= ? AND timestamp <= ?
	`, inClause)
	args = []any{}
	for _, name := range targetNames {
		args = append(args, name)
	}
	args = append(args, time.Unix(start, 0), time.Unix(end, 0))
	s.chConn.QueryRow(ctx, query, args...).Scan(&errorCount)
	status.TotalErrors = int64(errorCount)

	// 获取最后同步时间
	var lastSync time.Time
	query = fmt.Sprintf(`
		SELECT max(timestamp)
		FROM sync_events
		WHERE target_name IN (%s)
	`, inClause)
	args = []any{}
	for _, name := range targetNames {
		args = append(args, name)
	}
	s.chConn.QueryRow(ctx, query, args...).Scan(&lastSync)
	status.LastSyncTime = lastSync

	return status, nil
}

// expandTargetNames 扩展目标名称，用于兼容历史数据
func (s *Store) expandTargetNames(targetName string) []string {
	names := []string{targetName}

	// 解析 "Type:name" 格式
	if idx := strings.Index(targetName, ":"); idx > 0 {
		name := targetName[idx+1:]
		names = append(names, name)

		// 添加可能的变体
		// clickhouse-main -> clickhouse-primary
		if strings.HasPrefix(name, "clickhouse-") {
			names = append(names, "clickhouse-primary")
		}
	}

	return names
}

// Cleanup 清理过期数据
func (s *Store) Cleanup(ctx context.Context, before int64) error {
	if s.chConn == nil {
		return nil
	}

	beforeTime := time.Unix(before, 0)

	// 清理事件
	if err := s.chConn.Exec(ctx, "ALTER TABLE sync_events DELETE WHERE timestamp < ?", beforeTime); err != nil {
		return err
	}

	// 清理错误
	if err := s.chConn.Exec(ctx, "ALTER TABLE sync_errors DELETE WHERE timestamp < ?", beforeTime); err != nil {
		return err
	}

	// 清理位置
	if err := s.chConn.Exec(ctx, "ALTER TABLE sync_positions DELETE WHERE timestamp < ?", beforeTime); err != nil {
		return err
	}

	// 清理指标
	if err := s.chConn.Exec(ctx, "ALTER TABLE sync_metrics DELETE WHERE timestamp < ?", beforeTime); err != nil {
		return err
	}

	return nil
}

// boolToInt 布尔转整数
func boolToInt(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
