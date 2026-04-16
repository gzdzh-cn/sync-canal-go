# MySQL 到多目标同步方案

使用 [go-mysql](https://github.com/go-mysql-org/go-mysql) 实现 MySQL Binlog 增量数据同步，支持同时同步到 ClickHouse、Elasticsearch、MySQL 等多个目标。

![Dashboard](https://raw.githubusercontent.com/gzdzh-cn/sync-canal-admin/main/README/dashboard.png)

## 仓库地址

- **前端仓库**: https://github.com/gzdzh-cn/sync-canal-admin
- **后端仓库**: https://github.com/gzdzh-cn/sync-canal-go

## 目录

- [方案介绍](#方案介绍)
- [环境要求](#环境要求)
- [快速开始](#快速开始)
- [部署方式](#部署方式)
- [配置说明](#配置说明)
- [代码说明](#代码说明)
- [监控 API](#监控-api)
- [性能优化](#性能优化)
- [故障排查](#故障排查)
- [扩展新目标](#扩展新目标)

---

## 方案介绍

### 为什么选择 go-mysql?

| 特性 | go-mysql | Canal |
|------|----------|-------|
| MySQL 8.4 支持  | ✅ | ⚠️ |
| 多目标同步 | ✅ 自定义 | ⚠️ 需配置 |
| 依赖 | 无 | Java+ZK |
| 灵活性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| 性能 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |

### 工作原理

```
MySQL (Master)
    ↓ Binlog (ROW 模式)
go-mysql Canal (伪装成 Slave)
    ↓ 解析事件
MultiHandler (事件广播)
    ├→ ClickHouse Target (数据同步)
    │      ↓
    │   ClickHouse (目标数据库)
    │
    └→ Collector (监控采集)
           ↓
        RingBuffer (内存缓冲)
           ↓
        Store → ClickHouse (监控数据)
           ↓
        HTTP API (查询展示)
```

### 核心功能

- ✅ **多目标同步**: 支持同时同步到 ClickHouse/ES/MySQL 等多个目标
- ✅ **Canal 模式**: 伪装成 MySQL 从库,监听 binlog
- ✅ **增量同步**: 实时同步 INSERT/UPDATE/DELETE 操作（禁用 mysqldump）
- ✅ **指定表同步**: 每个目标可独立配置要同步的表
- ✅ **断点续传**: 保存 binlog 位置,支持断点恢复
- ✅ **MySQL 8.x**: 完全支持 MySQL 8.0/8.4
- ✅ **定时优化**: 自动执行 OPTIMIZE 清理 ReplacingMergeTree 重复数据
- ✅ **完整监控**: 实时 QPS/TPS、延迟追踪、事件历史、SSE 推送
- ✅ **HTTP API**: 20+ 监控接口，支持 Swagger 文档

### 技术栈

| 类别 | 技术 | 版本 | 说明 |
|------|------|------|------|
| **语言** | Go | 1.23+ | 主开发语言 |
| **框架** | GoFrame | v2 | Web 框架、配置管理、日志 |
| **Binlog 解析** | go-mysql | latest | Canal 模式监听 MySQL binlog |
| **目标存储** | ClickHouse | 24.3+ | 数据同步目标 |
| **监控存储** | ClickHouse | 24.3+ | 监控数据持久化（可选） |
| **ClickHouse 客户端** | clickhouse-go | v2 | ClickHouse Go 驱动 |
| **API 文档** | OpenAPI/Swagger | - | 自动生成 API 文档 |

**架构特点：**
- 采用 GoFrame v2 标准项目结构，分层清晰
- 接口化设计，易于扩展新的同步目标
- 监控与同步解耦，可独立启用/禁用
- 支持内存模式和 ClickHouse 持久化两种监控模式

---

## 环境要求

### 软件要求

- Go 1.23+
- MySQL 8.4+ (开启 Binlog ROW 模式)
- ClickHouse 24.3+ (如使用 ClickHouse 目标)

### MySQL 配置要求

```ini
# /etc/mysql/mysql.conf.d/mysqld.cnf
[mysqld]
server-id = 1
log_bin = mysql-bin
binlog_format = ROW
binlog_row_image = FULL
gtid_mode = ON
enforce_gtid_consistency = ON
binlog_expire_logs_seconds = 604800
```

### 验证 MySQL 配置

```sql
mysql -h 127.0.0.1 -P 13308 -u root -p

SHOW VARIABLES LIKE 'log_bin';           # 必须为 ON
SHOW VARIABLES LIKE 'binlog_format';     # 必须为 ROW
SHOW VARIABLES LIKE 'binlog_row_image';  # 必须为 FULL
```

---

## 快速开始

### 1. 项目结构

```
sync-canal-go/
├── main.go                    # 主程序入口
├── internal/
│   ├── cmd/                   # 命令入口
│   │   └── cmd.go            # 启动流程
│   ├── consts/                # 常量定义
│   ├── controller/            # HTTP 控制器
│   │   └── monitor/          # 监控 API
│   │       ├── monitor.go    # 监控控制器（20+ API）
│   │       └── routes.go     # 路由注册
│   ├── logic/                 # 业务逻辑层
│   │   ├── monitor/          # 监控模块
│   │   │   ├── collector.go  # 指标采集器
│   │   │   ├── store.go      # ClickHouse 存储
│   │   │   ├── ringbuffer.go # 环形缓冲区
│   │   │   └── tables.sql    # 建表 SQL
│   │   └── sync/             # 同步模块
│   │       ├── sync.go       # 同步服务
│   │       ├── handler.go    # 多目标事件处理器
│   │       └── clickhouse.go # ClickHouse 目标实现
│   ├── model/                 # 数据模型
│   │   └── entity/           # 实体定义
│   │       ├── config.go     # 配置实体
│   │       └── monitor.go    # 监控实体
│   └── service/               # 服务接口
│       ├── sync.go           # 同步接口
│       ├── monitor.go        # 监控接口
│       └── clickhouse.go     # ClickHouse 服务
├── manifest/
│   └── config/
│       └── config.yaml       # 配置文件
├── hack/                      # 开发工具
├── logs/                      # 日志目录
├── bin/                       # 编译输出
├── Makefile                   # 构建脚本
├── go.mod
└── go.sum
```

### 2. 安装依赖

```bash
cd sync-canal-go
go mod tidy
```

### 3. 创建 MySQL 同步账号

```sql
mysql -h 127.0.0.1 -P 13308 -u root -p

CREATE USER 'sync_user'@'%' IDENTIFIED BY 'sync_password_123';
GRANT SELECT, RELOAD, SHOW DATABASES, REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'sync_user'@'%';
FLUSH PRIVILEGES;
```

### 4. 配置同步规则

编辑 `manifest/config/config.yaml`:

```yaml
# HTTP 服务配置
server:
  address: ":8000"
  openapiPath: "/api.json"
  swaggerPath: "/swagger"

# 监控配置
monitor:
  enabled: true           # 是否启用监控
  historyDays: 30         # 历史数据保留天数
  collectPeriod: 10       # 采集周期(秒)
  maxEvents: 10000        # 内存最大事件数
  maxErrors: 1000         # 内存最大错误数
  clickhouse:             # 监控数据存储
    host: "127.0.0.1"
    port: 8124
    user: "default"
    password: "dzh123456"
    database: "sync_monitor"

# 日志配置
logger:
  level: "all"
  stdout: true
  path: "./logs"
  file: "sync-{Y-m-d}.log"
  rotate: "daily"
  backups: 30

# 同步源配置
database:
  default:
    link: "mysql:sync_user:sync_password_123@tcp(127.0.0.1:13308)/dzh3136_go"
    debug: true

# Canal 配置
canal:
  serverId: 1001
  flavor: "mysql"
  dump: false             # 仅增量同步

# 同步目标配置
sync:
  database: "dzh3136_go"
  batchSize: 1000
  targets:
    - name: "clickhouse-main"
      type: "clickhouse"
      tables:
        - "addons_customer_pro_clues"
      clickhouse:
        host: "127.0.0.1"
        port: 8124
        user: "default"
        password: "dzh123456"
        database: "dzh3136_go"
      schedule:            # 定时 OPTIMIZE 清理
        enable: true
        interval: 1440     # 分钟，1440 = 24小时
```

### 5. 创建 ClickHouse 表

ClickHouse 表需要手动创建，使用 `ReplacingMergeTree` 引擎：

```sql
-- 连接 ClickHouse
clickhouse-client -h 127.0.0.1 --port 9000 -u default --password dzh123456

-- 创建数据库
CREATE DATABASE IF NOT EXISTS dzh3136_go;

-- 创建表（字段与 MySQL 一致，外加 _version）
CREATE TABLE IF NOT EXISTS dzh3136_go.addons_customer_pro_clues (
    id String,
    createTime DateTime,
    updateTime DateTime,
    deleted_at Nullable(DateTime),
    serialId Nullable(Int64),
    name Nullable(String),
    -- ... 其他字段 ...
    _version Int64
) ENGINE = ReplacingMergeTree(_version)
ORDER BY id
SETTINGS index_granularity = 8192;
```

**引擎说明：**
- 使用 `ReplacingMergeTree(_version)` 引擎，`_version` 作为版本列
- 查询时需加 `FINAL` 关键字获取最新版本数据
- 相同 `id` 的记录会保留 `_version` 最大的版本

### 6. 启动同步程序

#### 方式一：从 Release 下载运行

```bash
# 下载对应平台的 Release 包（以 Linux amd64 为例）
wget https://github.com/gzdzh-cn/sync-canal-go/releases/download/v3.0.0/sync-canal-go_3.0.0_linux_amd64.tar.gz

# 解压
tar -xzf sync-canal-go_3.0.0_linux_amd64.tar.gz

# 进入目录
cd sync-canal-go_3.0.0_linux_amd64

# 修改配置文件
vim manifest/config/config.yaml

# 运行
./sync-canal-go
```

#### 方式二：从源码编译运行

```bash
# 下载仓库到本地
git clone https://github.com/gzdzh-cn/sync-canal-go.git
cd sync-canal-go

# 本地编译
go build -o bin/sync-canal-go .

# 或使用 GoFrame 命令，一键生成多平台版本（配置文件在 hack/config.yaml）
gf build

# 运行 （以 Linux amd64 为例）
./bin/v3.0.0/linux_amd64/sync-canal-go

# 或直接运行
go run main.go
```

### 7. 访问监控界面

启动后可通过以下方式访问监控：

```bash
# 仪表盘 浏览器打开
http://localhost:8000/monitor/dashboard

# Swagger 文档
open http://localhost:8000/swagger

# 服务状态
curl http://localhost:8000/monitor/status

# 健康检查
curl http://localhost:8000/monitor/health

# 实时推送（SSE）
curl http://localhost:8000/monitor/status/realtime
```

**监控前端界面展示：**

| 仪表盘 | 事件监控 |
|--------|----------|
| ![Dashboard](https://raw.githubusercontent.com/gzdzh-cn/sync-canal-admin/main/README/dashboard.png) | ![Events](https://raw.githubusercontent.com/gzdzh-cn/sync-canal-admin/main/README/events.png) |

| 延迟分析 | 错误排查 |
|----------|----------|
| ![Latency](https://raw.githubusercontent.com/gzdzh-cn/sync-canal-admin/main/README/latency.png) | ![Errors](https://raw.githubusercontent.com/gzdzh-cn/sync-canal-admin/main/README/errors.png) |

| 历史轨迹 | 配置管理 |
|----------|----------|
| ![History](https://raw.githubusercontent.com/gzdzh-cn/sync-canal-admin/main/README/history.png) | ![Config](https://raw.githubusercontent.com/gzdzh-cn/sync-canal-admin/main/README/config.png) |

### 8. 验证同步

```bash
# 在 MySQL 中更新数据
mysql -h 127.0.0.1 -P 3306 -u root -p dzh3136_go -e "UPDATE addons_customer_pro_clues SET wechat = 'TEST_SYNC' WHERE id = 'xxx'"

# 等待 2 秒后查询 ClickHouse
curl -s "http://127.0.0.1:8123/?user=default&password=dzh123456" \
  --data "SELECT id, wechat FROM dzh3136_go.addons_customer_pro_clues FINAL WHERE id = 'xxx'"
```

---

## 部署方式

### 方式一: 从 Git 仓库部署

```bash
# 克隆仓库
git clone https://github.com/gzdzh-cn/sync-canal-go.git

# 进入项目目录
cd sync-canal-go

# 安装依赖
go mod tidy

# 修改配置文件
vim manifest/config/config.yaml

# 直接运行
go run .

# 或编译后运行
go build -o bin/sync-canal-go .
./bin/sync-canal-go
```

### 方式二: Systemd 服务

```bash
# 从仓库克隆
git clone https://github.com/gzdzh-cn/sync-canal-go.git
cd sync-canal-go

# 编译程序
go build -o /opt/sync-canal-go/sync-canal-go .

# 复制配置文件
cp -r manifest/config /opt/sync-canal-go/

# 创建 systemd 服务
cat > /etc/systemd/system/sync-canal-go.service << EOF
[Unit]
Description=MySQL to Multi-Target Sync Service
After=network.target mysql.service clickhouse.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/sync-canal-go
ExecStart=/opt/sync-canal-go/sync-canal-go
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
systemctl daemon-reload
systemctl start sync-canal-go
systemctl enable sync-canal-go

# 查看状态
systemctl status sync-canal-go
```

### 方式三: Docker 部署

```bash
# 从仓库克隆
git clone https://github.com/gzdzh-cn/sync-canal-go.git
cd sync-canal-go

# 构建 Docker 镜像
docker build -t sync-canal-go .

# 运行容器
docker run -d --name sync-canal-go \
  -v /path/to/config:/app/manifest/config \
  -p 8000:8000 \
  sync-canal-go

# 构建多平台镜像并推送
docker buildx build --platform linux/amd64,linux/arm64 \
  --tag registry.cn-heyuan.aliyuncs.com/gzdzh/sync-canal-go:v3.0.0 \
  --push .
```

---

## 配置说明

### HTTP 服务配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| server.address | 监听地址 | :8000 |
| server.openapiPath | OpenAPI 路径 | /api.json |
| server.swaggerPath | Swagger 路径 | /swagger |

### 监控配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| monitor.enabled | 是否启用监控 | true |
| monitor.historyDays | 历史数据保留天数 | 30 |
| monitor.collectPeriod | 采集周期(秒) | 10 |
| monitor.maxEvents | 内存最大事件数 | 10000 |
| monitor.maxErrors | 内存最大错误数 | 1000 |

### MySQL 配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| database.default.link | MySQL 连接字符串 | - |

### Canal 配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| canal.serverId | 服务器 ID (唯一) | 1001 |
| canal.flavor | MySQL 类型 | mysql |
| canal.dump | 是否全量同步 | false |

### 同步目标配置 (targets)

每个目标独立配置，支持 `clickhouse`、`elasticsearch`、`mysql` 类型：

| 参数 | 说明 | 适用类型 |
|------|------|----------|
| name | 目标名称（唯一标识） | 全部 |
| type | 目标类型 | 全部 |
| tables | 该目标同步的表列表 | 全部 |
| schedule.enable | 启用定时 OPTIMIZE | clickhouse |
| schedule.interval | OPTIMIZE 间隔(分钟) | clickhouse |

### ClickHouse 目标配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| clickhouse.host | ClickHouse 地址 | 127.0.0.1 |
| clickhouse.port | 端口（支持 HTTP 8123/8124 和 TCP 9000） | 8124 |
| clickhouse.user | 用户名 | default |
| clickhouse.password | 密码 | - |
| clickhouse.database | 数据库 | - |

### Elasticsearch 目标配置 (预留)

| 参数 | 说明 | 默认值 |
|------|------|--------|
| elasticsearch.hosts | ES 节点地址列表 | - |
| elasticsearch.username | 用户名 | - |
| elasticsearch.password | 密码 | - |
| elasticsearch.index | 索引前缀（留空用表名） | - |

### MySQL 目标配置 (预留)

| 参数 | 说明 | 默认值 |
|------|------|--------|
| mysql_target.host | 目标 MySQL 地址 | - |
| mysql_target.port | 目标 MySQL 端口 | 3306 |
| mysql_target.user | 用户名 | - |
| mysql_target.password | 密码 | - |
| mysql_target.database | 目标数据库 | - |

### 全局同步配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| sync.database | 源数据库 | - |
| sync.batch_size | 批量大小 | 1000 |

### 日志配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| level | 日志级别 | info |
| file | 日志文件路径 | ./logs/sync.log |
| stdout | 输出到控制台 | true |
| rotate | 分割方式 | size |
| max_size | 单文件最大(MB) | 100 |
| naming | 文件命名规则 | date |

---

## 代码说明

### 架构设计

项目采用 **GoFrame v2** 标准项目结构，分层清晰：

```
┌─────────────────────────────────────────────────┐
│                   main.go                        │
│                      ↓                           │
│                   cmd.go                         │
│            ┌─────────┴─────────┐                │
│            ↓                   ↓                 │
│      sync/ (同步)        monitor/ (监控)         │
│            ↓                   ↓                 │
│      MultiHandler         Collector             │
│            ↓                   ↓                 │
│      SyncTarget           IStore                │
│            ↓                   ↓                 │
│      ClickHouse          ClickHouse             │
└─────────────────────────────────────────────────┘
```

### 核心模块

#### 1. 同步模块 (`internal/logic/sync`)

| 文件 | 功能 |
|------|------|
| `sync.go` | 同步服务核心：配置加载、Canal 创建、目标创建 |
| `handler.go` | MultiHandler：将事件广播到多个目标，集成监控采集 |
| `clickhouse.go` | ClickHouse 目标实现：INSERT/UPDATE/DELETE 处理，定时 OPTIMIZE |

**关键特性：**
- 支持多目标同步（MultiHandler 广播机制）
- 仅增量同步（禁用 mysqldump）
- 自动过滤表（IncludeTableRegex）
- 定时 OPTIMIZE 清理（可配置间隔）

#### 2. 监控模块 (`internal/logic/monitor`)

| 文件 | 功能 |
|------|------|
| `collector.go` | 指标采集器：QPS/TPS 计算、事件/错误/位置采集 |
| `store.go` | ClickHouse 存储：历史数据持久化、统计查询 |
| `ringbuffer.go` | 环形缓冲区：内存中保留最近 N 条事件/错误 |

**监控指标：**
- **实时指标**：QPS、TPS、延迟秒数
- **事件统计**：INSERT/UPDATE/DELETE 数量、平均耗时
- **延迟统计**：当前/平均/最大/最小/P95/P99 延迟
- **错误统计**：按级别/目标/表分组统计

### 数据流程

```
MySQL Binlog
    ↓
  Canal (go-mysql)
    ↓
  MultiHandler
    ├── ClickHouseTarget (同步数据)
    │      ↓
    │   ClickHouse (目标数据库)
    │
    └── Collector (监控采集)
           ↓
        RingBuffer (内存)
           ↓
        Store → ClickHouse (监控数据)
           ↓
        HTTP API (查询展示)
```

### 关键接口

```go
// SyncTarget 同步目标接口
type SyncTarget interface {
    Connect(ctx context.Context) error  // 连接目标
    Close() error                       // 关闭连接
    OnRow(e *canal.RowsEvent) error     // 处理行变更
    OnDDL(...) error                    // 处理 DDL
    Start()                             // 启动后台任务
    String() string                     // 目标名称
}

// ICollector 采集器接口
type ICollector interface {
    OnEvent(e *entity.SyncEvent)        // 采集事件
    OnPosition(p *entity.SyncPosition)  // 采集位置
    OnError(err *entity.SyncError)      // 采集错误
    RegisterTarget(name, targetType string)
    UpdateTargetStatus(name, status string)
}

// IStore 存储接口
type IStore interface {
    SaveEvent(ctx, e) error             // 保存事件
    SavePosition(ctx, p) error          // 保存位置
    SaveError(ctx, err) error           // 保存错误
    GetEventStats(ctx, start, end)      // 获取统计
    GetLatencyStats(ctx, start, end)    // 获取延迟
    // ...
}
```

---

## 监控 API

### API 列表

| 路径 | 方法 | 功能 |
|------|------|------|
| `/monitor/status` | GET | 服务状态（QPS/TPS/延迟/binlog位置） |
| `/monitor/health` | GET | 健康检查 |
| `/monitor/metrics` | GET | 当前指标 |
| `/monitor/metrics/history` | GET | 指标历史 |
| `/monitor/events` | GET | 事件列表（分页、过滤） |
| `/monitor/events/stats` | GET | 事件统计 |
| `/monitor/errors` | GET | 错误列表 |
| `/monitor/errors/stats` | GET | 错误统计 |
| `/monitor/position` | GET | 当前同步位置 |
| `/monitor/position/history` | GET | 位置历史 |
| `/monitor/latency` | GET | 延迟统计 |
| `/monitor/latency/history` | GET | 延迟历史 |
| `/monitor/targets` | GET | 目标列表 |
| `/monitor/targets/{name}/enable` | POST | 启用目标 |
| `/monitor/targets/{name}/disable` | POST | 禁用目标 |
| `/monitor/config` | GET | 获取配置 |
| `/monitor/config` | POST | 更新配置 |
| `/monitor/status/realtime` | GET | SSE 实时推送 |
| `/monitor/stream` | GET | SSE 实时推送（别名） |

### 使用示例

```bash
# 获取服务状态
curl http://localhost:8000/monitor/status

# 获取事件统计（最近7天）
curl http://localhost:8000/monitor/events/stats

# 获取延迟统计（最近1小时）
curl "http://localhost:8000/monitor/latency?startTime=$(($(date +%s) - 3600))"

# 查询事件列表
curl "http://localhost:8000/monitor/events?tableName=addons_customer_pro_clues&page=1&pageSize=20"

# 实时推送（SSE）
curl -N http://localhost:8000/monitor/status/realtime
```

### 监控数据表

监控数据存储在 ClickHouse 的 `sync_monitor` 数据库：

```sql
-- 监控指标表
sync_metrics (
    timestamp DateTime,
    metric_name String,
    metric_value Float64,
    target_name String,
    table_name String,
    tags Map(String, String)
) TTL timestamp + INTERVAL 30 DAY

-- 同步事件表
sync_events (
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
) TTL timestamp + INTERVAL 30 DAY

-- 同步位置表
sync_positions (
    timestamp DateTime,
    target_name String,
    binlog_file String,
    binlog_pos UInt64,
    gtid String,
    delay_seconds UInt32
) TTL timestamp + INTERVAL 30 DAY

-- 错误日志表
sync_errors (
    timestamp DateTime,
    level String,
    target_name String,
    table_name String,
    message String,
    stack_trace String,
    raw_data String
) TTL timestamp + INTERVAL 30 DAY
```

---

## 性能优化

### 批量写入

```go
// 使用 ClickHouse 批量插入
batch, _ := chConn.PrepareBatch(ctx, "INSERT INTO table VALUES")
for _, row := range rows {
    batch.Append(row...)
}
batch.Send()
```

### ClickHouse 优化

```sql
-- 异步插入
SET async_insert = 1;

-- 等待异步插入
SET wait_for_async_insert = 0;
```

### ReplacingMergeTree 优化

程序会自动定时执行 `OPTIMIZE TABLE` 清理重复数据：
- 在每个 ClickHouse target 中通过 `schedule.enable` 开启/关闭
- 通过 `schedule.interval` 设置间隔（默认 24 小时）

### 监控性能优化

- **环形缓冲区**：固定大小，避免内存无限增长
- **异步写入**：监控数据异步写入 ClickHouse，不阻塞同步
- **TTL 过期**：自动清理过期历史数据
- **批量写入**：定时批量写入，减少 IO

---

## 故障排查

### 连接 MySQL 失败

```bash
# 检查 MySQL 是否运行
docker ps | grep mysql

# 测试连接
mysql -h 127.0.0.1 -P 13308 -u sync_user -p

# 检查权限
SHOW GRANTS FOR 'sync_user'@'%';
```

### Binlog 格式错误

```sql
-- 检查配置
SHOW VARIABLES LIKE 'binlog_format';  -- 必须为 ROW

-- 修改配置
SET GLOBAL binlog_format = 'ROW';
```

### ClickHouse 连接失败

```bash
# 检查 ClickHouse 是否运行
curl http://127.0.0.1:8123/ping

# 检查端口
lsof -i :8124
```

### 列数不匹配

确保 ClickHouse 表结构与 MySQL 一致（外加 `_version` 列）。

### 断点丢失

程序会自动保存 binlog 位置，重启后从断点继续同步。

### 某个目标写入失败

MultiHandler 中某个目标失败只记录错误日志，不影响其他目标和 Canal 继续运行。

### 监控数据丢失

- 检查 `monitor.enabled` 是否为 `true`
- 检查 ClickHouse 连接是否正常
- 查看日志中的错误信息

---

## 扩展新目标

要添加新的同步目标类型（如 Elasticsearch），只需：

1. 在 `internal/model/entity/config.go` 中添加对应的配置结构
2. 在 `internal/logic/sync/` 目录新建文件（如 `elasticsearch.go`）
3. 实现 `SyncTarget` 接口
4. 在 `sync.go` 的 `CreateTargets` 工厂函数中注册新类型

```go
// elasticsearch.go
type ElasticsearchTarget struct { ... }

func NewElasticsearchTarget(tc *entity.TargetConfig, sync *entity.SyncConfig) (*ElasticsearchTarget, error) { ... }
func (t *ElasticsearchTarget) Connect(ctx context.Context) error { ... }
func (t *ElasticsearchTarget) Close() error { ... }
func (t *ElasticsearchTarget) OnRow(e *canal.RowsEvent) error { ... }
func (t *ElasticsearchTarget) OnDDL(...) error { ... }
func (t *ElasticsearchTarget) Start() { ... }
func (t *ElasticsearchTarget) String() string { ... }
```

---

## 服务管理

```bash
# 查看进程
ps aux | grep sync-canal-go

# 停止服务
pkill -f sync-canal-go
# 或
systemctl stop sync-canal-go

# 查看日志
tail -f logs/sync-{Y-m-d}.log
# 或
journalctl -u sync-canal-go -f

# 使用 Makefile
make build        # 编译
make run          # 运行
make docker-build # 构建 Docker 镜像
```

---

## 注意事项

1. **ClickHouse 表使用 ReplacingMergeTree 引擎**，查询时需要加 `FINAL` 关键字获取最新数据
2. **每个目标独立配置表列表**，只有目标中配置的表才会被同步
3. **Canal 层面过滤**取所有目标表的并集，未配置的表不会被监听
4. **某个目标失败不影响其他目标**，MultiHandler 会继续处理
5. **MySQL 需要开启 Binlog ROW 模式**
6. **监控数据会占用 ClickHouse 存储空间**，建议配置 TTL
7. **监控可选**，如果 ClickHouse 连接失败会降级为内存模式

---

## 相关文档

- [go-mysql GitHub](https://github.com/go-mysql-org/go-mysql)
- [go-mysql 文档](https://pkg.go.dev/github.com/go-mysql-org/go-mysql)
- [ClickHouse 客户端](https://github.com/ClickHouse/clickhouse-go)
- [GoFrame 文档](https://goframe.org)

---

## 更新记录

| 日期 | 版本 | 说明 |
|------|------|------|
| 2026-04-13 | v1.0 | 初始版本（单 ClickHouse 目标） |
| 2026-04-15 | v1.1 | 合并文档，更新项目结构 |
| 2026-04-15 | v2.0 | 多目标架构重构，支持同时同步到 ClickHouse/ES/MySQL；精简 main() |
| 2026-04-16 | v3.0 | 完整监控系统：实时 QPS/TPS、延迟追踪、事件历史、20+ HTTP API、SSE 推送；GoFrame v2 标准项目结构 |
