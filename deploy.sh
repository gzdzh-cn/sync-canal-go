#!/bin/bash

# go-mysql 同步方案部署脚本

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# ========================================
# 配置变量
# ========================================
PROJECT_DIR="/Volumes/disk/site/go/dzhgo/dzhgo-admin"
SYNC_DIR="${PROJECT_DIR}/cmd"
INSTALL_DIR="/opt/mysql_to_clickhouse"

# ========================================
# 检查环境
# ========================================
check_environment() {
    log_info "检查环境..."

    # 检查 Go
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装"
        log_info "安装方法: brew install go"
        exit 1
    fi
    log_info "✓ Go 版本: $(go version)"

    # 检查 MySQL 连接
    log_info "检查 MySQL 连接..."
    if ! nc -z 127.0.0.1 13308 2>/dev/null; then
        log_warn "MySQL 端口 13308 不可达"
    else
        log_info "✓ MySQL 连接正常"
    fi

    # 检查 ClickHouse 连接
    log_info "检查 ClickHouse 连接..."
    if ! curl -s "http://127.0.0.1:8123/ping" | grep -q "Ok"; then
        log_warn "ClickHouse 端口 8123 不可达"
    else
        log_info "✓ ClickHouse 连接正常"
    fi
}

# ========================================
# 安装依赖
# ========================================
install_dependencies() {
    log_info "安装依赖..."

    cd ${PROJECT_DIR}

    # 安装 go-mysql
    log_info "安装 go-mysql..."
    go get github.com/go-mysql-org/go-mysql

    # 安装 ClickHouse 客户端
    log_info "安装 ClickHouse 客户端..."
    go get github.com/ClickHouse/clickhouse-go/v2

    # 安装 YAML 解析库
    log_info "安装 YAML 解析库..."
    go get gopkg.in/yaml.v3

    log_info "✓ 依赖安装完成"
}

# ========================================
# 创建同步程序
# ========================================
create_sync_program() {
    log_info "创建同步程序..."

    # 创建目录
    mkdir -p ${SYNC_DIR}

    # 复制文件
    if [ -f "main.go" ]; then
        cp main.go ${SYNC_DIR}/
        log_info "✓ main.go 已复制"
    else
        log_error "main.go 不存在"
        exit 1
    fi

    if [ -f "config.yaml" ]; then
        cp config.yaml ${SYNC_DIR}/
        log_info "✓ config.yaml 已复制"
    else
        log_error "config.yaml 不存在"
        exit 1
    fi

    log_info "✓ 同步程序创建完成"
}

# ========================================
# 编译程序
# ========================================
build_program() {
    log_info "编译程序..."

    cd ${PROJECT_DIR}

    # 编译
    go build -o bin/mysql_to_clickhouse cmd/main.go

    if [ $? -eq 0 ]; then
        log_info "✓ 编译成功: bin/mysql_to_clickhouse"
    else
        log_error "✗ 编译失败"
        exit 1
    fi
}

# ========================================
# 安装服务
# ========================================
install_service() {
    log_info "安装服务..."

    # 创建安装目录
    mkdir -p ${INSTALL_DIR}/{bin,conf,logs,data}

    # 复制文件
    cp ${PROJECT_DIR}/bin/mysql_to_clickhouse ${INSTALL_DIR}/bin/
    cp ${SYNC_DIR}/config.yaml ${INSTALL_DIR}/conf/

    # 创建 systemd 服务
    cat > /etc/systemd/system/mysql-to-clickhouse.service << EOF
[Unit]
Description=MySQL to ClickHouse Sync Service
After=network.target mysql.service clickhouse.service

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/bin/mysql_to_clickhouse -config ${INSTALL_DIR}/conf/config.yaml
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    # 重载 systemd
    systemctl daemon-reload

    log_info "✓ 服务安装完成"
    log_info "启动命令: systemctl start mysql-to-clickhouse"
}

# ========================================
# 创建 MySQL 同步账号
# ========================================
create_mysql_user() {
    log_info "创建 MySQL 同步账号..."

    read -p "MySQL root 密码: " -s MYSQL_PASSWORD
    echo

    mysql -h 127.0.0.1 -P 13308 -u root -p${MYSQL_PASSWORD} << 'EOF'
-- 创建同步账号
CREATE USER IF NOT EXISTS 'sync_user'@'%' IDENTIFIED BY 'sync_password_123';

-- 授予权限
GRANT SELECT, RELOAD, SHOW DATABASES, REPLICATION SLAVE, REPLICATION CLIENT
ON *.* TO 'sync_user'@'%';

FLUSH PRIVILEGES;

-- 验证
SHOW GRANTS FOR 'sync_user'@'%';
EOF

    if [ $? -eq 0 ]; then
        log_info "✓ MySQL 同步账号创建成功"
    else
        log_error "✗ MySQL 同步账号创建失败"
        exit 1
    fi
}

# ========================================
# 测试同步
# ========================================
test_sync() {
    log_info "测试同步..."

    # 启动服务
    log_info "启动服务..."
    cd ${PROJECT_DIR}
    ./bin/mysql_to_clickhouse -config cmd/config.yaml &
    SYNC_PID=$!

    # 等待启动
    sleep 5

    # 插入测试数据
    log_info "插入测试数据..."
    read -p "MySQL root 密码: " -s MYSQL_PASSWORD
    echo

    mysql -h 127.0.0.1 -P 13308 -u root -p${MYSQL_PASSWORD} dzh3136_go << 'EOF'
INSERT INTO clues (id, guest_id, name, mobile, create_time, update_time)
VALUES ('test_deploy_001', 'guest_001', '部署测试', '13900139000', NOW(), NOW());
EOF

    # 等待同步
    sleep 3

    # 验证 ClickHouse
    log_info "验证 ClickHouse..."
    if curl -s "http://127.0.0.1:8123/?user=default&password=dzh123456"
        --data "SELECT COUNT(*) FROM dzh3136_go.clues WHERE id = 'test_deploy_001'" | grep -q "1"; then
        log_info "✓ 同步测试成功"
    else
        log_error "✗ 同步测试失败"
    fi

    # 清理测试数据
    mysql -h 127.0.0.1 -P 13308 -u root -p${MYSQL_PASSWORD} dzh3136_go
        -e "DELETE FROM clues WHERE id = 'test_deploy_001'"

    # 停止服务
    kill $SYNC_PID 2>/dev/null
}

# ========================================
# 主菜单
# ========================================
show_menu() {
    echo ""
    echo "========================================"
    echo " go-mysql 同步方案部署脚本"
    echo "========================================"
    echo "1. 完整安装 (推荐)"
    echo "2. 检查环境"
    echo "3. 安装依赖"
    echo "4. 创建同步程序"
    echo "5. 编译程序"
    echo "6. 安装服务"
    echo "7. 创建 MySQL 同步账号"
    echo "8. 测试同步"
    echo "0. 退出"
    echo "========================================"
    read -p "请选择 [0-8]: " choice

    case $choice in
        1)
            check_environment
            install_dependencies
            create_sync_program
            build_program
            create_mysql_user
            log_info "✓ 完整安装完成!"
            log_info "运行测试: ./deploy.sh (选择 8)"
            ;;
        2)
            check_environment
            ;;
        3)
            install_dependencies
            ;;
        4)
            create_sync_program
            ;;
        5)
            build_program
            ;;
        6)
            install_service
            ;;
        7)
            create_mysql_user
            ;;
        8)
            test_sync
            ;;
        0)
            log_info "退出"
            exit 0
            ;;
        *)
            log_error "无效选择"
            ;;
    esac
}

# ========================================
# 主程序
# ========================================
main() {
    # 检查是否为 root
    if [ "$EUID" -ne 0 ]; then
        log_warn "建议使用 root 用户执行此脚本"
    fi

    # 显示菜单
    while true; do
        show_menu
    done
}

# 执行主程序
main
