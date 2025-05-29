# Kong网关DNS服务发现系统

一个基于DNS的服务发现系统，专为Kong网关设计，提供服务自动注册、健康检查和DNS解析功能。

## 功能特点

- 基于DNS协议的服务发现
- 服务自动注册和注销
- 健康检查和故障检测
- Web管理界面
- 多端口安全隔离设计
- 与Kong网关无缝集成

## 系统架构

本系统采用三端口设计:
- **注册端口 (8080)**: 服务实例注册、注销和心跳
- **管理端口 (9090)**: 管理API和监控功能
- **DNS端口 (53)**: 标准DNS服务

## 快速开始

### 准备工作

- 安装Go 1.21+
- 安装etcd 3.5+

### 启动服务

```bash
# 克隆代码
git clone https://github.com/hewenyu/kong-discovery.git
cd kong-discovery

# 编译
go build -o kong-discovery cmd/discovery/main.go

# 启动
./kong-discovery
```

### Docker 部署

```bash
# 使用Docker Compose启动所有服务
docker-compose up -d
```

## 技术栈

- Go语言
- Echo框架
- etcd存储
- React前端

## 许可证

MIT
