# Releases 下载

## v3.0.0 (2026-04-16)

### 功能特性

- ✅ 多目标同步：支持同时同步到 ClickHouse/ES/MySQL
- ✅ 完整监控系统：实时 QPS/TPS、延迟追踪、事件历史
- ✅ 20+ HTTP API、SSE 实时推送
- ✅ GoFrame v2 标准项目结构

### 下载地址

| 平台 | 架构 | 下载链接 |
|------|------|----------|
| **Linux** | amd64 | [sync-canal-go_v3.0.0_linux_amd64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v3.0.0/sync-canal-go_v3.0.0_linux_amd64.tar.gz) |
| **Linux** | arm64 | [sync-canal-go_v3.0.0_linux_arm64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v3.0.0/sync-canal-go_v3.0.0_linux_arm64.tar.gz) |
| **macOS** | amd64 | [sync-canal-go_v3.0.0_darwin_amd64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v3.0.0/sync-canal-go_v3.0.0_darwin_amd64.tar.gz) |
| **macOS** | arm64 (M1/M2) | [sync-canal-go_v3.0.0_darwin_arm64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v3.0.0/sync-canal-go_v3.0.0_darwin_arm64.tar.gz) |
| **Windows** | amd64 | [sync-canal-go_v3.0.0_windows_amd64.zip](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v3.0.0/sync-canal-go_v3.0.0_windows_amd64.zip) |
| **Windows** | arm64 | [sync-canal-go_v3.0.0_windows_arm64.zip](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v3.0.0/sync-canal-go_v3.0.0_windows_arm64.zip) |

### 安装说明

#### Linux / macOS

```bash
# 下载（以 Linux amd64 为例）
wget https://github.com/gzdzh-cn/sync-canal-go/releases/download/v3.0.0/sync-canal-go_v3.0.0_linux_amd64.tar.gz

# 解压
tar -xzf sync-canal-go_v3.0.0_linux_amd64.tar.gz

# 进入目录
cd sync-canal-go

# 修改配置
vim manifest/config/config.yaml

# 运行
./sync-canal-go
```

#### Windows

```powershell
# 下载并解压 sync-canal-go_v3.0.0_windows_amd64.zip

# 进入目录，修改配置文件 manifest/config/config.yaml

# 运行
.\sync-canal-go.exe
```

---

## v2.0.0 (2026-04-15)

### 功能特性

- ✅ 多目标架构重构，支持同时同步到 ClickHouse/ES/MySQL
- ✅ 精简 main()，代码结构优化

### 下载地址

| 平台 | 架构 | 下载链接 |
|------|------|----------|
| **Linux** | amd64 | [sync-canal-go_v2.0.0_linux_amd64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v2.0.0/sync-canal-go_v2.0.0_linux_amd64.tar.gz) |
| **Linux** | arm64 | [sync-canal-go_v2.0.0_linux_arm64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v2.0.0/sync-canal-go_v2.0.0_linux_arm64.tar.gz) |
| **macOS** | amd64 | [sync-canal-go_v2.0.0_darwin_amd64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v2.0.0/sync-canal-go_v2.0.0_darwin_amd64.tar.gz) |
| **macOS** | arm64 | [sync-canal-go_v2.0.0_darwin_arm64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v2.0.0/sync-canal-go_v2.0.0_darwin_arm64.tar.gz) |
| **Windows** | amd64 | [sync-canal-go_v2.0.0_windows_amd64.zip](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v2.0.0/sync-canal-go_v2.0.0_windows_amd64.zip) |
| **Windows** | arm64 | [sync-canal-go_v2.0.0_windows_arm64.zip](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v2.0.0/sync-canal-go_v2.0.0_windows_arm64.zip) |

---

## v1.1.0 (2026-04-15)

### 功能特性

- ✅ 合并文档，更新项目结构

### 下载地址

| 平台 | 架构 | 下载链接 |
|------|------|----------|
| **Linux** | amd64 | [sync-canal-go_v1.1.0_linux_amd64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v1.1.0/sync-canal-go_v1.1.0_linux_amd64.tar.gz) |
| **macOS** | amd64 | [sync-canal-go_v1.1.0_darwin_amd64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v1.1.0/sync-canal-go_v1.1.0_darwin_amd64.tar.gz) |
| **Windows** | amd64 | [sync-canal-go_v1.1.0_windows_amd64.zip](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v1.1.0/sync-canal-go_v1.1.0_windows_amd64.zip) |

---

## v1.0.0 (2026-04-13)

### 功能特性

- ✅ 初始版本（单 ClickHouse 目标）

### 下载地址

| 平台 | 架构 | 下载链接 |
|------|------|----------|
| **Linux** | amd64 | [sync-canal-go_v1.0.0_linux_amd64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v1.0.0/sync-canal-go_v1.0.0_linux_amd64.tar.gz) |
| **macOS** | amd64 | [sync-canal-go_v1.0.0_darwin_amd64.tar.gz](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v1.0.0/sync-canal-go_v1.0.0_darwin_amd64.tar.gz) |
| **Windows** | amd64 | [sync-canal-go_v1.0.0_windows_amd64.zip](https://github.com/gzdzh-cn/sync-canal-go/releases/download/v1.0.0/sync-canal-go_v1.0.0_windows_amd64.zip) |

---

## 发布说明

### Release 打包内容

每个 Release 包含：

```
sync-canal-go_vX.X.X_平台_架构.tar.gz
├── sync-canal-go          # 可执行文件
├── manifest/
│   └── config/
│       └── config.yaml    # 配置文件
└── README.md              # 说明文档
```

### 构建命令

```bash
# 克隆仓库
git clone https://github.com/gzdzh-cn/sync-canal-go.git
cd sync-canal-go

# 多平台构建
gf build

# 打包发布文件
make release VERSION=v3.0.0
```

### 校验文件

每个版本附带校验文件：

- `checksums.txt` - SHA256 校验和
- `checksums.txt.sig` - GPG 签名（可选）
