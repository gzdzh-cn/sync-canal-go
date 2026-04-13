# go-mysql 同步方案快速入门

本指南帮助你在 10 分钟内完成 MySQL 8.4 到 ClickHouse 的增量同步配置。

## 前置条件

- ✅ Go 1.16+ 已安装
- ✅ MySQL 8.4 已安装并运行
- ✅ ClickHouse 已安装并运行
- ✅ MySQL 已开启 Binlog (ROW 模式)

## 快速开始

### 1. 检查 MySQL Binlog 配置

```bash
# 连接 MySQL
mysql -h 127.0.0.1 -P 13308 -u root -p

# 检查配置
SHOW VARIABLES LIKE 'log_bin';           # 必须为 ON
SHOW VARIABLES LIKE 'binlog_format';     # 必须为 ROW
SHOW VARIABLES LIKE 'binlog_row_image';  # 必须为 FULL
```

### 2. 创建 MySQL 同步账号

```sql
CREATE USER 'sync_user'@'%' IDENTIFIED BY 'sync_password_123';
GRANT SELECT, RELOAD, SHOW DATABASES, REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'sync_user'@'%';
FLUSH PRIVILEGES;
```

### 3. 配置同步程序

编辑 `cmd/config.yaml`:

```yaml
sync:
  database: dzh3136_go
  # 只同步这里指定的表，其他表忽略
  tables:
    - addons_customer_pro_clues
```

### 4. 手动创建 ClickHouse 表

```bash
# 连接 ClickHouse
clickhouse-client -h 127.0.0.1 --port 9000 -u default --password dzh123456
```

```sql
-- 创建数据库（如不存在）
CREATE DATABASE IF NOT EXISTS dzh3136_go;

-- 创建线索表（字段与 MySQL 完全一致，外加 _version）
CREATE TABLE IF NOT EXISTS dzh3136_go.addons_customer_pro_clues (
    id String,
    createTime DateTime,
    updateTime DateTime,
    deleted_at Nullable(DateTime),
    serialId Nullable(Int64),
    name Nullable(String),
    created_name Nullable(String),
    created_id Nullable(String),
    account_id Nullable(String),
    guest_id Nullable(String),
    project_id Nullable(String),
    services_id Nullable(String),
    services_ids Nullable(String),
    mobile Nullable(String),
    wechat Nullable(String),
    weixin Nullable(String),
    source_from Nullable(String),
    keywords Nullable(String),
    followup_type Nullable(String),
    last_followup_time Nullable(DateTime),
    level Nullable(String),
    ocean_time Nullable(DateTime),
    allot_time Nullable(DateTime),
    remark Nullable(String),
    orderNum Int32 DEFAULT 99,
    status Nullable(Int64),
    filterRemark Nullable(String),
    filter_group_ids Nullable(String),
    dtype Nullable(Int32),
    _version Int64
) ENGINE = ReplacingMergeTree(_version)
ORDER BY id
SETTINGS index_granularity = 8192;
```

**字段说明：**

| 字段 | MySQL 类型 | ClickHouse 类型 | 说明 |
|------|-----------|-----------------|------|
| id | varchar(191) | String | 主键 |
| createTime | datetime(3) | DateTime | 创建时间 |
| updateTime | datetime(3) | DateTime | 更新时间 |
| deleted_at | datetime(3) | Nullable(DateTime) | 软删除标记 |
| serialId | int | Nullable(Int64) | 序列号 |
| name | varchar(200) | Nullable(String) | 姓名 |
| created_name | varchar(64) | Nullable(String) | 创建者名称 |
| created_id | varchar(191) | Nullable(String) | 创建者ID |
| account_id | varchar(64) | Nullable(String) | 账户ID |
| guest_id | varchar(64) | Nullable(String) | 访客ID |
| project_id | varchar(64) | Nullable(String) | 项目ID |
| services_id | varchar(191) | Nullable(String) | 客服ID |
| services_ids | varchar(500) | Nullable(String) | 客服IDs |
| mobile | varchar(32) | Nullable(String) | 手机号 |
| wechat | varchar(128) | Nullable(String) | 微信号 |
| weixin | varchar(128) | Nullable(String) | 微信号2 |
| source_from | varchar(32) | Nullable(String) | 来源 |
| keywords | varchar(500) | Nullable(String) | 关键词 |
| followup_type | varchar(191) | Nullable(String) | 跟进类型 |
| last_followup_time | datetime(3) | Nullable(DateTime) | 最后跟进时间 |
| level | varchar(100) | Nullable(String) | 等级 |
| ocean_time | datetime(3) | Nullable(DateTime) | 公海时间 |
| allot_time | datetime(3) | Nullable(DateTime) | 分配时间 |
| remark | text | Nullable(String) | 备注 |
| orderNum | int | Int32 | 排序号 |
| status | bigint | Nullable(Int64) | 状态 |
| filterRemark | text | Nullable(String) | 筛选备注 |
| filter_group_ids | varchar(500) | Nullable(String) | 筛选组IDs |
| dtype | int | Nullable(Int32) | 数据类型 |
| _version | - | Int64 | 版本号（ReplacingMergeTree 必需） |

**引擎说明：**
- 使用 `ReplacingMergeTree(_version)` 引擎，`_version` 作为版本列
- 查询时需加 `FINAL` 关键字获取最新版本数据
- 相同 `id` 的记录会保留 `_version` 最大的版本

**关系表查询说明：**
- 筛选条件（客服、等级、跟进类型、筛选组）通过 MySQL 关系表实时查询
- 关系表：`addons_customer_pro_clues_service`、`addons_customer_pro_clues_level`、`addons_customer_pro_clues_followup_type`、`addons_customer_pro_clues_filter_group`

### 5. 导入全量数据

首次同步需要先导入全量数据:

```bash
cd /Volumes/disk/site/go/dzhgo/dzhgo-admin
go run cmd/import_to_clickhouse.go
```

### 6. 启动同步程序

```bash
cd /Volumes/disk/site/go/dzhgo/dzhgo-admin/clickhouse/go-mysql

# 编译
go build -o bin/mysql_to_clickhouse ./cmd/main.go

# 运行
./bin/mysql_to_clickhouse -config ./cmd/config.yaml

# 后台运行
nohup ./bin/mysql_to_clickhouse -config ./cmd/config.yaml > logs/sync.log 2>&1 &
```

### 7. 验证同步

```bash
# 1. 在 MySQL 中更新数据
mysql -h 127.0.0.1 -P 13308 -u root -p dzh3136_go -e "UPDATE addons_customer_pro_clues SET wechat = 'TEST_SYNC' WHERE id = 'xxx'"

# 2. 等待 2 秒后查询 ClickHouse (使用 FINAL 获取最新版本)
curl -s "http://127.0.0.1:8123/?user=default&password=dzh123456" --data "SELECT id, wechat FROM dzh3136_go.addons_customer_pro_clues FINAL WHERE id = 'xxx'"
```

## 服务管理

```bash
# 查看进程
ps aux | grep mysql_to_clickhouse

# 停止服务
pkill -f mysql_to_clickhouse

# 查看日志
tail -f logs/sync.log
```

## 注意事项

1. **ClickHouse 表使用 ReplacingMergeTree 引擎**，查询时需要加 `FINAL` 关键字获取最新数据
2. **只同步配置中指定的表**，其他表的变更会被忽略
3. **数组字段**（services_ids_list, followup_type_list 等）需要通过定时任务刷新

## 故障排查

### 连接 MySQL 失败

```bash
# 检查 MySQL 是否运行
docker ps | grep mysql

# 测试连接
mysql -h 127.0.0.1 -P 13308 -u sync_user -p
```

### ClickHouse 连接失败

```bash
# 检查 ClickHouse 是否运行
curl http://127.0.0.1:8123/ping

# 检查端口占用
lsof -i :8123
```

### 列数不匹配

确保 ClickHouse 表结构与 MySQL 一致（外加数组字段和 _version 列）。

### 打包docker镜像

```bash
# 构建 Docker 镜像
docker build -t mysql-to-clickhouse .

# 运行 Docker 容器
docker run -d --name mysql-to-clickhouse -v /path/to/config.yaml:/app/cmd/config.yaml mysql-to-clickhouse

# 构建多平台
docker buildx build --platform linux/amd64,linux/arm64 --tag registry.cn-heyuan.aliyuncs.com/gzdzh/mysql_to_clickhouse:v1.0.0 --push .

GOWORK=off go mod vendor && docker buildx build --platform linux/amd64,linux/arm64 --tag registry.cn-heyuan.aliyuncs.com/gzdzh/mysql_to_clickhouse:v1.0.0 --push .
```