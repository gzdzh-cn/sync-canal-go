# GitHub Actions 部署教程

本文档介绍如何使用 GitHub Actions 自动化构建、部署 sync-canal-go 项目。

## 目录

- [概述](#概述)
- [前置要求](#前置要求)
- [配置 GitHub Secrets](#配置-github-secrets)
- [服务器准备](#服务器准备)
- [部署流程](#部署流程)
- [手动操作](#手动操作)
- [常见问题](#常见问题)

---

## 概述

### 工作流说明

| 工作流文件 | 触发条件 | 功能 |
|-----------|---------|------|
| `ci.yml` | push/PR 到 main/develop | 代码检查、构建、测试 |
| `docker.yml` | push 到 main 或创建 tag | 构建 Docker 镜像并推送到阿里云 |
| `deploy.yml` | 手动触发 或 docker.yml 完成 | SSH 到服务器部署容器 |
| `release.yml` | 创建 tag `v*` | 多平台二进制打包发布到 GitHub Release |

### 部署流程图

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  push 代码   │ →  │ CI 构建测试  │ →  │ Docker 构建  │ →  │ SSH 部署     │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
       ↓
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  创建 tag    │ →  │ 多平台打包   │ →  │ GitHub Release│
└─────────────┘    └─────────────┘    └─────────────┘
```

### 镜像仓库

- **阿里云镜像仓库**: `registry.cn-heyuan.aliyuncs.com/gzdzh/sync-canal-go`
- **镜像标签**: `latest` (主分支) 或 `v1.0.0` (版本标签)

---

## 前置要求

### 1. 服务器要求

- Docker 已安装
- 开放端口: 8000 (服务端口)、SSH 端口
- 可访问阿里云镜像仓库

### 2. 本地要求

- Git 已配置
- 有 GitHub 仓库的 push 权限
- 有阿里云镜像仓库的访问权限

### 3. 检查服务器 Docker

```bash
# SSH 到服务器
ssh root@你的服务器IP

# 检查 Docker
docker --version

# 登录阿里云镜像仓库
docker login registry.cn-heyuan.aliyuncs.com
```

---

## 配置 GitHub Secrets

在 GitHub 仓库页面：**Settings → Secrets and variables → Actions → New repository secret**

### 必填变量 (5个)

| Secret 名称 | 说明 | 获取方式 |
|------------|------|---------|
| `ALIYUN_REGISTRY_USERNAME` | 阿里云镜像仓库用户名 | 阿里云控制台 → 容器镜像服务 |
| `ALIYUN_REGISTRY_PASSWORD` | 阿里云镜像仓库密码 | 阿里云控制台 → 容器镜像服务 → 访问凭证 |
| `SERVER_HOST` | 服务器 IP 地址 | 如 `192.168.1.100` 或域名 |
| `SERVER_USER` | SSH 用户名 | 如 `root` |
| `SERVER_SSH_KEY` | SSH 私钥 | 见下方获取方式 |

### 可选变量 (2个)

| Secret 名称 | 说明 | 默认值 |
|------------|------|--------|
| `SERVER_PORT` | SSH 端口 | `22` |
| `DEPLOY_PATH` | 服务器部署目录 | `/www/wwwroot/docker/sync-canal-go` |

### 获取 SSH 私钥

```bash
# 查看本地私钥
cat ~/.ssh/id_rsa

# 如果没有私钥，生成一对
ssh-keygen -t rsa -b 4096 -C "github-actions"

# 将公钥添加到服务器
ssh-copy-id -i ~/.ssh/id_rsa.pub root@你的服务器IP
```

### 配置截图示例

```
Settings
  └── Secrets and variables
        └── Actions
              ├── New repository secret
              │     ├── Name: ALIYUN_REGISTRY_USERNAME
              │     └── Value: your-username
              │
              ├── New repository secret
              │     ├── Name: ALIYUN_REGISTRY_PASSWORD
              │     └── Value: your-password
              │
              ├── New repository secret
              │     ├── Name: SERVER_HOST
              │     └── Value: 192.168.1.100
              │
              ├── New repository secret
              │     ├── Name: SERVER_USER
              │     └── Value: root
              │
              └── New repository secret
                    ├── Name: SERVER_SSH_KEY
                    └── Value: -----BEGIN RSA PRIVATE KEY-----
                               MIIEpAIBAAKCAQEA...
                               -----END RSA PRIVATE KEY-----
```

---

## 服务器准备

### 1. 创建部署目录

```bash
# SSH 到服务器
ssh root@你的服务器IP

# 创建目录 (使用默认路径)
mkdir -p /www/wwwroot/docker/sync-canal-go/logs
mkdir -p /www/wwwroot/docker/sync-canal-go/config

# 或使用自定义路径 (需要配置 DEPLOY_PATH Secret)
mkdir -p /your/custom/path/logs
mkdir -p /your/custom/path/config
```

### 2. 准备配置文件

```bash
cd /www/wwwroot/docker/sync-canal-go

# 创建配置文件 (从项目复制并修改)
vim config/config.yaml
```

**config.yaml 示例** (从项目复制并修改数据库连接等敏感信息):

```yaml
# https://goframe.org/docs/web/server-config-file-template
server:
  address:     ":8000"
  openapiPath: "/api.json"
  swaggerPath: "/swagger"
  serverRoot:  "resource/public"

# 监控配置
monitor:
  enabled:       true    # 是否启用监控
  historyDays:   30      # 历史数据保留天数
  collectPeriod: 10      # 采集周期(秒)
  maxEvents:     10000   # 内存最大事件数
  maxErrors:     1000    # 内存最大错误数
  # ClickHouse 监控数据存储(可选，留空则仅使用内存)
  clickhouse:
    host:     "127.0.0.1"
    port:     8124
    user:     "default"
    password: "your_password"
    database: "sync_monitor"

# https://goframe.org/docs/core/glog-config
logger:
  level:   "all"
  stdout:  true
  path:    "/app/logs"
  file:    "sync-{Y-m-d}.log"
  rotate:  "daily"
  backups: 30

# 同步源配置
database:
  default:
    link: "mysql:sync_user:your_password@tcp(127.0.0.1:3306)/your_database"
    debug: true

# Canal 配置
canal:
  serverId: 1001
  flavor:   "mysql"
  # 仅增量同步，禁用 mysqldump
  dump: false

# 同步目标配置
sync:
  # 源数据库
  database: "your_database"
  # 批量大小
  batchSize: 1000
  # 同步目标列表
  targets:
    - name: "clickhouse-main"
      type: "clickhouse"
      tables:
        - "your_table_name"
      clickhouse:
        host:     "127.0.0.1"
        port:     8124
        user:     "default"
        password: "your_password"
        database: "your_database"
      # 定时 OPTIMIZE 清理
      schedule:
        enable:   true
        interval: 1440  # 分钟，1440 = 24小时
```

> **注意**: 请根据实际环境修改数据库连接信息、表名等配置。完整配置参考 `manifest/config/config.yaml`。

### 3. 登录镜像仓库

```bash
docker login registry.cn-heyuan.aliyuncs.com
# 输入用户名和密码
```

---

## 部署流程

### 自动部署

推送代码到 main 分支，自动触发完整部署流程：

```bash
# 提交代码
git add .
git commit -m "feat: 新功能"
git push origin main
```

**流程**:
1. CI 工作流运行 (代码检查、构建、测试)
2. Docker 工作流构建镜像并推送
3. Deploy 工作流 SSH 到服务器部署

### 发布版本

创建 tag 触发版本发布：

```bash
# 创建 tag
git tag v1.0.0
git push origin v1.0.0
```

**流程**:
1. Docker 工作流构建 `v1.0.0` 镜像
2. Release 工作流打包多平台二进制
3. 创建 GitHub Release 页面

### 手动部署

在 GitHub 页面手动触发：

1. 进入 **Actions** 页面
2. 选择 **Deploy** 工作流
3. 点击 **Run workflow**
4. 输入镜像标签 (如 `v1.0.0` 或 `latest`)
5. 点击 **Run workflow**

---

## 手动操作

### 查看部署状态

```bash
# GitHub Actions 页面
https://github.com/你的用户名/你的仓库/actions
```

### 查看服务器容器

```bash
# SSH 到服务器
ssh root@你的服务器IP

# 查看容器状态
docker ps | grep sync-canal-go

# 查看日志
docker logs -f sync-canal-go

# 查看最近 100 行日志
docker logs --tail 100 sync-canal-go
```

### 手动重启容器

```bash
docker restart sync-canal-go
```

### 手动更新镜像

```bash
# 拉取最新镜像
docker pull registry.cn-heyuan.aliyuncs.com/gzdzh/sync-canal-go:latest

# 停止旧容器
docker stop sync-canal-go
docker rm sync-canal-go

# 启动新容器
docker run -d \
  --name sync-canal-go \
  --restart always \
  -p 8000:8000 \
  -v /www/wwwroot/docker/sync-canal-go/config/config.yaml:/app/manifest/config/config.yaml:ro \
  -v /www/wwwroot/docker/sync-canal-go/logs:/app/logs \
  registry.cn-heyuan.aliyuncs.com/gzdzh/sync-canal-go:latest
```

### 回滚到指定版本

```bash
# 使用指定版本镜像
docker pull registry.cn-heyuan.aliyuncs.com/gzdzh/sync-canal-go:v1.0.0

docker stop sync-canal-go
docker rm sync-canal-go

docker run -d \
  --name sync-canal-go \
  --restart always \
  -p 8000:8000 \
  -v /www/wwwroot/docker/sync-canal-go/config/config.yaml:/app/manifest/config/config.yaml:ro \
  -v /www/wwwroot/docker/sync-canal-go/logs:/app/logs \
  registry.cn-heyuan.aliyuncs.com/gzdzh/sync-canal-go:v1.0.0
```

---

## 常见问题

### Q1: Docker 登录失败

**错误**: `unauthorized: authentication required`

**解决**:
```bash
# 确认 Secrets 配置正确
# ALIYUN_REGISTRY_USERNAME 和 ALIYUN_REGISTRY_PASSWORD

# 本地测试登录
docker login registry.cn-heyuan.aliyuncs.com
```

### Q2: SSH 连接失败

**错误**: `Permission denied (publickey)`

**解决**:
```bash
# 1. 检查 SERVER_SSH_KEY Secret 是否正确配置
# 2. 确保私钥格式正确 (包含完整的 BEGIN 和 END 行)
# 3. 确保公钥已添加到服务器的 ~/.ssh/authorized_keys

# 在服务器上检查
cat ~/.ssh/authorized_keys
```

### Q3: 容器启动失败

**错误**: 容器反复重启

**解决**:
```bash
# 查看容器日志
docker logs sync-canal-go

# 常见原因:
# 1. 配置文件路径不正确
# 2. 数据库连接失败
# 3. 端口被占用

# 检查配置文件
ls -la /www/wwwroot/docker/sync-canal-go/config/config.yaml

# 检查端口
netstat -tlnp | grep 8000
```

### Q4: 镜像拉取失败

**错误**: `Error: image gzdzh/sync-canal-go:latest not found`

**解决**:
```bash
# 确认镜像已推送
# 检查 GitHub Actions 是否成功完成

# 手动拉取测试
docker pull registry.cn-heyuan.aliyuncs.com/gzdzh/sync-canal-go:latest
```

### Q5: 配置文件修改后不生效

**解决**:
```bash
# 修改配置文件后重启容器
vim /www/wwwroot/docker/sync-canal-go/config/config.yaml
docker restart sync-canal-go
```

### Q6: 如何查看部署历史

```bash
# GitHub Actions 页面
https://github.com/你的用户名/你的仓库/actions

# 点击具体的工作流运行记录查看详细日志
```

---

## 附录

### 目录结构

```
sync-canal-go/
├── .github/
│   └── workflows/
│       ├── ci.yml          # CI 工作流
│       ├── docker.yml      # Docker 构建推送
│       ├── deploy.yml      # SSH 部署
│       └── release.yml     # 版本发布
├── Dockerfile              # 多阶段构建
├── docker-compose.yml      # 本地/服务器部署示例
└── docs/
    └── GITHUB_ACTIONS_DEPLOY.md  # 本文档
```

### 服务器目录结构

```
/www/wwwroot/docker/sync-canal-go/
├── config/
│   └── config.yaml    # 配置文件（挂载到容器 /app/manifest/config/config.yaml）
└── logs/              # 日志目录（挂载到容器 /app/logs）
    └── sync-{Y-m-d}.log
```

### 配置文件挂载说明

容器内配置文件路径: `/app/manifest/config/config.yaml`

部署时通过 Docker volume 挂载:
```bash
-v /www/wwwroot/docker/sync-canal-go/config/config.yaml:/app/manifest/config/config.yaml:ro
```

这样服务器上的 `/www/wwwroot/docker/sync-canal-go/config/config.yaml` 会覆盖容器内的默认配置。

### 相关链接

- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [Docker 文档](https://docs.docker.com/)
- [阿里云容器镜像服务](https://cr.console.aliyun.com/)
