# 构建阶段
FROM --platform=$BUILDPLATFORM registry.cn-heyuan.aliyuncs.com/gzdzh/golang:1.21-alpine AS builder

# 设置 Go 代理（使用国内代理）
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

# 设置工作目录
WORKDIR /build

# 声明构建参数
ARG TARGETOS
ARG TARGETARCH

# 复制 go.mod 和 go.sum
COPY go.mod go.sum* ./

# 复制 vendor 目录
COPY vendor/ vendor/

# 复制源代码
COPY main.go ./main.go

# 编译（使用 vendor，支持多平台）
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -mod=vendor -ldflags="-w -s" -o sync-service .

# 运行阶段
FROM registry.cn-heyuan.aliyuncs.com/gzdzh/alpine:latest

# 设置时区 (使用 POSIX 格式，无需 tzdata 文件)
ENV TZ=CST-8

# 设置工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /build/sync-service .

# 创建配置和日志目录
RUN mkdir -p /app/config /app/logs
COPY config/config.yaml ./config/config.yaml

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep sync-service || exit 1

# 启动服务
CMD ["./sync-service"]
