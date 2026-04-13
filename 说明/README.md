# go-mysql 同步方案部署教程

使用 [go-mysql](https://github.com/go-mysql-org/go-mysql) 实现 MySQL 8.4 到 ClickHouse 的增量数据同步。

## 目录

- [方案介绍](#方案介绍)
- [环境要求](#环境要求)
- [部署步骤](#部署步骤)
- [代码说明](#代码说明)
- [配置说明](#配置说明)
- [运行管理](#运行管理)
- [故障排查](#故障排查)

---

## 方案介绍

### 为什么选择 go-mysql?

| 特性 | go-mysql | go-mysql-transfer | Canal |
|------|----------|------------------|-------|
| MySQL 8.4 支持 | ✅ | ✅ | ⚠️ |
| ClickHouse 支持 | ✅ 自定义 | ❌ | ⚠️ 需配置 |
| 依赖 | 无 | 无 | Java+ZK |
| 灵活性 | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ |
| 性能 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |

### 工作原理

```
MySQL 8.4 (Master)
    ↓ Binlog (ROW 模式)
go-mysql Canal (伪装成 Slave)
    ↓ 解析事件
自定义 Handler
    ↓ 转换数据
ClickHouse
```

### go-mysql 功能

- ✅ **Canal 模式**: 伪装成 MySQL 从库,监听 binlog
- ✅ **全量+增量**: 支持 dump 全量数据 + binlog 增量同步
- ✅ **指定表同步**: 可选择特定数据库和表
- ✅ **GTID 支持**: 支持 GTID 复制
- ✅ **断点续传**: 保存 binlog 位置,支持断点恢复
- ✅ **MySQL 8.x**: 完全支持 MySQL 8.0/8.4

---

## 环境要求

### 软件要求

- Go 1.16+
- MySQL 8.4+ (开启 Binlog ROW 模式)
- ClickHouse 24.3+

### MySQL 配置要求

```ini
# /etc/mysql/mysql.conf.d/mysqld.cnf
[mysqld]
# 服务器唯一 ID
server-id = 1

# 开启 binlog
log_bin = mysql-bin

# 必须为 ROW 模式
binlog_format = ROW

# 完整行镜像
binlog_row_image = FULL

# GTID 模式(推荐)
gtid_mode = ON
enforce_gtid_consistency = ON

# binlog 保留时间
binlog_expire_logs_seconds = 604800
```

### 验证 MySQL 配置

```sql
-- 连接 MySQL
mysql -h 127.0.0.1 -P 13308 -u root -p

-- 检查 binlog 配置
SHOW VARIABLES LIKE 'log_bin';           # ON
SHOW VARIABLES LIKE 'binlog_format';     # ROW
SHOW VARIABLES LIKE 'binlog_row_image';  # FULL
SHOW VARIABLES LIKE 'server_id';         # 非 0
```

---

## 部署步骤

### 1. 安装依赖

```bash
# 进入项目目录
cd /Volumes/disk/site/go/dzhgo/dzhgo-admin/clickhouse/go-mysql

# 安装
go mod tidy
```

### 2. 创建同步程序

```bash

```

### 3. 配置同步规则

编辑 `cmd/mysql_to_clickhouse/config.yaml`:

```yaml
# MySQL 配置
mysql:
  host: 127.0.0.1
  port: 13308
  user: sync_user
  password: sync_password_123
  server_id: 1001

# ClickHouse 配置
clickhouse:
  host: 127.0.0.1
  port: 9000
  user: default
  password: dzh123456
  database: dzh3136_go

# 同步规则
sync:
  database: dzh3136_go
  tables:
    - clues
  # 是否全量同步
  full_sync: false
```

### 4. 创建 MySQL 同步账号

```sql
-- 连接 MySQL
mysql -h 127.0.0.1 -P 13308 -u root -p

-- 创建同步账号
CREATE USER 'sync_user'@'%' IDENTIFIED BY 'sync_password_123';

-- 授予权限
GRANT SELECT, RELOAD, SHOW DATABASES, REPLICATION SLAVE, REPLICATION CLIENT 
ON *.* TO 'sync_user'@'%';

FLUSH PRIVILEGES;
```

### 5. 初始化全量数据

首次同步需要先导入全量数据:

```bash
cd /Volumes/disk/site/go/dzhgo/dzhgo-admin

# 运行导入脚本
go run cmd/import_to_clickhouse.go
```

### 6. 启动同步程序

#### 方式一: 直接运行

```bash
cd /Volumes/disk/site/go/dzhgo/dzhgo-admin/clickhouse/go-mysql

# 运行同步程序
go run main.go

# 或编译后运行
go build -o bin/mysql_to_clickhouse main.go
./bin/mysql_to_clickhouse
```

#### 方式二: Systemd 服务

```bash
# 编译程序
cd /Volumes/disk/site/go/dzhgo/dzhgo-admin/clickhouse/go-mysql
go build -o /opt/mysql_to_clickhouse/sync main.go

# 创建 systemd 服务
cat > /etc/systemd/system/mysql-to-clickhouse.service << EOF
[Unit]
Description=MySQL to ClickHouse Sync Service
After=network.target mysql.service clickhouse.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/mysql_to_clickhouse
ExecStart=/opt/mysql_to_clickhouse/sync
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
systemctl daemon-reload
systemctl start mysql-to-clickhouse
systemctl enable mysql-to-clickhouse

# 查看状态
systemctl status mysql-to-clickhouse
```

### 7. 验证同步

```bash
# 1. 在 MySQL 中插入测试数据
mysql -h 127.0.0.1 -P 13308 -u root -p dzh3136_go << 'EOF'
INSERT INTO clues (id, guest_id, name, mobile, create_time, update_time) 
VALUES ('test_sync_001', 'guest_001', '同步测试', '13900139000', NOW(), NOW());
EOF

# 2. 等待 3 秒后查询 ClickHouse
sleep 3
curl "http://127.0.0.1:8123/?user=default&password=dzh123456" \
  --data "SELECT * FROM dzh3136_go.clues WHERE id = 'test_sync_001' FORMAT JSON"

# 3. 清理测试数据
mysql -h 127.0.0.1 -P 13308 -u root -p dzh3136_go \
  -e "DELETE FROM clues WHERE id = 'test_sync_001'"
```

---

## 代码说明

### 主要代码结构

```go
package main

import (
    "github.com/go-mysql-org/go-mysql/canal"
    "github.com/ClickHouse/clickhouse-go/v2"
)

// ClickHouseHandler 实现 canal.EventHandler 接口
type ClickHouseHandler struct {
    canal.DummyEventHandler
    chConn clickhouse.Conn
}

// OnRow 处理行变更事件
func (h *ClickHouseHandler) OnRow(e *canal.RowsEvent) error {
    switch e.Action {
    case canal.InsertAction:
        // 处理 INSERT
    case canal.UpdateAction:
        // 处理 UPDATE
    case canal.DeleteAction:
        // 处理 DELETE
    }
    return nil
}
```

### 关键接口

```go
// EventHandler 接口
type EventHandler interface {
    OnRow(e *RowsEvent) error
    OnRotate(e *RotateEvent) error
    OnDDL(nextPos Position, queryEvent *QueryEvent) error
    OnXID(nextPos Position) error
    OnGTID(gtid GTIDSet) error
    String() string
}
```

### 数据流程

1. **监听 Binlog**: Canal 伪装成 MySQL 从库
2. **解析事件**: 解析 INSERT/UPDATE/DELETE 事件
3. **转换数据**: 将 MySQL 数据转换为 ClickHouse 格式
4. **写入目标**: 批量写入 ClickHouse

---

## 配置说明

### MySQL 配置

```yaml
mysql:
  host: 127.0.0.1          # MySQL 地址
  port: 13308              # MySQL 端口
  user: sync_user          # 同步账号
  password: sync_password_123
  server_id: 1001          # 服务器 ID (唯一)
  flavor: mysql            # mysql 或 mariadb
```

### ClickHouse 配置

```yaml
clickhouse:
  host: 127.0.0.1          # ClickHouse 地址
  port: 9000               # Native 端口 (非 HTTP)
  user: default            # 用户名
  password: dzh123456      # 密码
  database: dzh3136_go     # 数据库
```

### 同步规则

```yaml
sync:
  database: dzh3136_go     # 源数据库
  tables:                  # 同步的表
    - clues
    - orders
  full_sync: false         # 是否全量同步
  batch_size: 1000         # 批量大小
```

---

## 运行管理

### 启动服务

```bash
# Systemd 方式
systemctl start mysql-to-clickhouse

# 查看状态
systemctl status mysql-to-clickhouse

# 查看日志
journalctl -u mysql-to-clickhouse -f
```

### 停止服务

```bash
systemctl stop mysql-to-clickhouse
```

### 重启服务

```bash
systemctl restart mysql-to-clickhouse
```

### 查看日志

```bash
# Systemd 日志
journalctl -u mysql-to-clickhouse -f

# 程序日志
tail -f /opt/mysql_to_clickhouse/logs/sync.log
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

### 并发处理

```yaml
sync:
  workers: 4           # 并发工作数
  batch_size: 1000     # 批量大小
  queue_size: 10000    # 队列大小
```

### ClickHouse 优化

```sql
-- 异步插入
SET async_insert = 1;

-- 等待异步插入
SET wait_for_async_insert = 0;
```

---

## 故障排查

### 问题 1: 连接 MySQL 失败

**症状:**
```
connect to mysql failed: Error 1045
```

**解决方案:**
```bash
# 检查账号权限
mysql -h 127.0.0.1 -P 13308 -u sync_user -p

# 重新授权
GRANT SELECT, RELOAD, SHOW DATABASES, REPLICATION SLAVE, REPLICATION CLIENT 
ON *.* TO 'sync_user'@'%';
FLUSH PRIVILEGES;
```

### 问题 2: Binlog 格式错误

**症状:**
```
binlog format must be ROW
```

**解决方案:**
```sql
-- 修改配置
SET GLOBAL binlog_format = 'ROW';
```

### 问题 3: ClickHouse 写入失败

**症状:**
```
insert to clickhouse failed
```

**解决方案:**
```bash
# 检查 ClickHouse 连接
curl http://127.0.0.1:8123/ping

# 检查表结构
curl "http://127.0.0.1:8123/?user=default&password=dzh123456" \
  --data "SHOW CREATE TABLE dzh3136_go.clues"
```

### 问题 4: 断点丢失

**解决方案:**
```bash
# 检查断点文件
ls -la /opt/mysql_to_clickhouse/data/

# 从指定位置启动
./sync -position="mysql-bin.000001:12345"
```

---

## 高级功能

### 指定表同步

```go
// 只监听指定表
cfg.IncludeTableRegex = []string{
    "dzh3136_go\\.clues",
    "dzh3136_go\\.orders",
}
```

### 过滤字段

```go
// 过滤不需要的字段
func filterColumns(row []interface{}, columns []*mysql.Column) []interface{} {
    filtered := make([]interface{}, 0)
    for i, col := range columns {
        if !isFiltered(col.Name) {
            filtered = append(filtered, row[i])
        }
    }
    return filtered
}
```

### 数据转换

```go
// 自定义数据转换
func convertValue(value interface{}, column *mysql.Column) interface{} {
    switch column.Type {
    case mysql.MYSQL_TYPE_DATETIME:
        // 转换时间格式
    case mysql.MYSQL_TYPE_JSON:
        // 处理 JSON
    }
    return value
}
```

---

## 相关文档

- [go-mysql GitHub](https://github.com/go-mysql-org/go-mysql)
- [go-mysql 文档](https://pkg.go.dev/github.com/go-mysql-org/go-mysql)
- [ClickHouse 客户端](https://github.com/ClickHouse/clickhouse-go)
- [ClickHouse 部署教程](../README.md)

---

## 更新记录

| 日期 | 版本 | 说明 |
|------|------|------|
| 2026-04-13 | v1.0 | 初始版本 |
