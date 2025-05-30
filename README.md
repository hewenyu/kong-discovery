# Kong网关DNS服务发现系统

基于DNS协议的服务发现系统，专为Kong网关设计，支持自动服务注册、健康检查和DNS解析。

## 功能特性

- 基于DNS协议的服务发现
- 自动化的服务注册和注销
- 服务健康检查和故障自动清理
- Web管理界面
- 多副本高可用部署支持
- 命名空间隔离

## 系统架构

系统由以下几个主要组件组成：

- **服务注册API** (端口8080)：处理服务实例的注册、注销和心跳
- **管理API** (端口9090)：提供系统管理功能
- **DNS服务** (端口53)：提供DNS解析服务
- **前端管理界面**：用于监控和管理服务
- **etcd**：用于存储服务信息和系统配置

## 快速开始

### 前置条件

- Docker 和 Docker Compose
- Go 1.21+ (仅开发环境需要)

### 本地开发环境启动

1. 克隆代码库

```bash
git clone https://github.com/hewenyu/kong-discovery.git
cd kong-discovery
```

2. 启动本地开发环境

```bash
# 使用Docker Compose启动etcd和DNS服务发现系统
cd deployments/docker-compose
docker-compose up -d

# 或者在本地启动etcd（需要自行安装）
docker run -d --name etcd -p 2379:2379 -p 2380:2380 bitnami/etcd:3.5 --allow-none-authentication
```

3. 编译并运行服务

```bash
# 编译
go build -o bin/kong-discovery ./cmd/dns-discovery

# 运行
./bin/kong-discovery --config configs/config.yaml
```

4. 测试DNS解析

```bash
# 在注册服务后，可以使用dig或nslookup测试DNS解析
dig @localhost service-name.default.service.local
```

## 配置说明

配置文件位于 `configs/config.yaml`，可以根据需要修改以下配置：

- DNS服务端口和协议
- 服务注册和管理API端口
- etcd连接信息
- 心跳间隔和超时时间
- 服务域名后缀

## 开发

详细的开发指南请参考 [docs/目录结构.md](docs/目录结构.md) 和 [docs/架构设计.md](docs/架构设计.md)。

### 运行测试

```bash
# 运行所有测试
go test -v ./...

# 运行特定模块测试
go test -v ./internal/store/etcd
```

## 部署

### Docker部署

参考 `deployments/docker-compose/docker-compose.yaml` 文件进行配置和部署。

### Kubernetes部署

参考 `deployments/kubernetes/` 目录下的配置文件进行Kubernetes部署。

## 贡献

欢迎提交Pull Request或Issue！

## 许可证

[MIT License](LICENSE)
