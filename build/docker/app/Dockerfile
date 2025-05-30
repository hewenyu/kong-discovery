FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache ca-certificates git

# 拷贝go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 拷贝源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kong-discovery ./cmd/dns-discovery

# 使用alpine作为最终镜像
FROM alpine:3.19

# 安装必要的运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非root用户
RUN adduser -D -H -h /app appuser

# 拷贝二进制文件和配置文件
COPY --from=builder /app/kong-discovery /app/
COPY configs /app/configs

# 设置工作目录
WORKDIR /app

# 使用非root用户运行
USER appuser

# 暴露服务端口
EXPOSE 53/udp 53/tcp 8080/tcp 9090/tcp

# 设置启动命令
ENTRYPOINT ["/app/kong-discovery"]
CMD ["--config", "/app/configs/config.yaml"] 