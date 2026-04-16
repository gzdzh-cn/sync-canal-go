-- =================================================================================
-- Sync Canal Monitor Tables
-- =================================================================================

-- 监控指标表
CREATE TABLE IF NOT EXISTS sync_metrics (
    timestamp DateTime64(3),
    metric_name String,
    metric_value Float64,
    target_name String,
    table_name String,
    tags Map(String, String)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (timestamp, metric_name)
TTL timestamp + INTERVAL 30 DAY;

-- 同步事件表
CREATE TABLE IF NOT EXISTS sync_events (
    timestamp DateTime64(3),
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
TTL timestamp + INTERVAL 30 DAY;

-- 同步位置表
CREATE TABLE IF NOT EXISTS sync_positions (
    timestamp DateTime64(3),
    target_name String,
    binlog_file String,
    binlog_pos UInt64,
    gtid String,
    delay_seconds UInt32
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (timestamp, target_name)
TTL timestamp + INTERVAL 30 DAY;

-- 错误日志表
CREATE TABLE IF NOT EXISTS sync_errors (
    timestamp DateTime64(3),
    level String,
    target_name String,
    table_name String,
    message String,
    stack_trace String,
    raw_data String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (timestamp, level)
TTL timestamp + INTERVAL 30 DAY;

-- =================================================================================
-- Indexes for better query performance
-- =================================================================================

-- 事件表索引
ALTER TABLE sync_events ADD INDEX IF NOT EXISTS idx_target_name target_name TYPE bloom_filter GRANULARITY 1;
ALTER TABLE sync_events ADD INDEX IF NOT EXISTS idx_table_name table_name TYPE bloom_filter GRANULARITY 1;
ALTER TABLE sync_events ADD INDEX IF NOT EXISTS idx_event_type event_type TYPE set(3) GRANULARITY 1;

-- 错误表索引
ALTER TABLE sync_errors ADD INDEX IF NOT EXISTS idx_error_level level TYPE set(2) GRANULARITY 1;
ALTER TABLE sync_errors ADD INDEX IF NOT EXISTS idx_error_target target_name TYPE bloom_filter GRANULARITY 1;

-- 位置表索引
ALTER TABLE sync_positions ADD INDEX IF NOT EXISTS idx_position_target target_name TYPE bloom_filter GRANULARITY 1;

-- =================================================================================
-- Materialized Views for Aggregation (Optional)
-- =================================================================================

-- 每小时事件统计物化视图
CREATE MATERIALIZED VIEW IF NOT EXISTS sync_events_hourly
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, target_name, table_name, event_type)
AS SELECT
    toStartOfHour(timestamp) AS hour,
    target_name,
    table_name,
    event_type,
    count() AS event_count,
    sum(rows_count) AS total_rows,
    avg(duration_ms) AS avg_duration,
    max(duration_ms) AS max_duration,
    sum(success) AS success_count
FROM sync_events
GROUP BY hour, target_name, table_name, event_type;

-- 每小时延迟统计物化视图
CREATE MATERIALIZED VIEW IF NOT EXISTS sync_latency_hourly
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, target_name)
AS SELECT
    toStartOfHour(timestamp) AS hour,
    target_name,
    avg(delay_seconds) AS avg_delay,
    max(delay_seconds) AS max_delay,
    min(delay_seconds) AS min_delay
FROM sync_positions
GROUP BY hour, target_name;
