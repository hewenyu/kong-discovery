# Kong Discovery 项目结构文档

## 项目概述

Kong Discovery 是一个为Kong网关提供服务发现功能的系统。它结合了DNS服务与etcd，实现了服务的自动注册与发现机制。

## 项目结构

```
kong-discovery/
├── cmd/                    # 应用入口点
│   └── main.go             # 主程序入口
├── configs/                # 配置文件目录
│   └── config.yaml         # 默认配置文件
├── doc/                    # 文档
│   ├── development_plan.md # 开发规划文档
│   └── project_structure.md # 本文档
├── internal/               # 内部包
│   ├── apihandler/         # API处理器模块
│   │   ├── handler.go      # API处理器接口和实现
│   │   └── handler_test.go # API处理器测试
│   ├── config/             # 配置管理模块
│   │   ├── config.go       # 配置结构和加载逻辑
│   │   ├── config_test.go  # 配置模块测试
│   │   ├── logger.go       # 日志接口和实现
│   │   └── logger_test.go  # 日志模块测试
│   ├── dnsserver/         # DNS服务器模块
│   │   ├── server.go      # DNS服务器接口和实现
│   │   └── server_test.go # DNS服务器测试
│   └── etcdclient/        # etcd客户端模块
│       ├── client.go      # etcd客户端接口和基本实现
│       ├── client_test.go # etcd客户端测试
│       └── service.go     # 服务发现相关功能实现
├── git.md                 # Git相关文档
├── go.mod                 # Go模块定义
├── go.sum                 # Go模块依赖校验和
├── kong-discovery         # 编译后的可执行文件
├── LICENSE                # 许可证文件
├── README.md              # 项目说明
└── todo.md                # 任务清单
```

## 当前开发进度

目前已完成的模块:

1. **项目初始化与基础结构** (任务1.1)
   - 初始化了Go模块 (`github.com/hewenyu/kong-discovery`)
   - 创建了基本目录结构

2. **日志模块** (任务1.2)
   - 实现了基于Zap的日志接口与实现
   - 支持开发环境和生产环境配置
   - 完成了单元测试

3. **配置管理模块** (任务1.3)
   - 使用Viper实现配置加载
   - 支持从文件和环境变量加载配置
   - 定义了完整的配置结构
   - 完成了单元测试

4. **etcd客户端模块** (任务1.4, 1.8, 2.1-2.4)
   - 实现了etcd连接与交互的接口
   - 实现了基本操作（连接、Ping、Get、GetWithPrefix）
   - 实现了DNS记录存储与查询功能
   - 实现了服务注册、注销和查询功能
   - 实现了服务实例到DNS记录的转换
   - 完成了集成测试

5. **API处理器模块** (任务1.5, 1.6)
   - 使用Echo框架搭建HTTP服务
   - 实现管理API端口监听与健康检查端点
   - 实现服务注册API端口监听与健康检查端点
   - 完成了单元测试

6. **DNS服务器模块** (任务1.7, 1.9)
   - 使用`miekg/dns`库实现基础DNS服务器
   - 支持硬编码DNS记录响应
   - 支持从etcd读取DNS记录
   - 支持服务发现DNS查询（A记录和SRV记录）
   - 完成了单元测试

## 下一步开发计划

即将开始开发的模块:

1. **DNS转发功能** (任务1.10)
   - 实现向上游DNS服务器转发未知域名的查询逻辑
   - 配置上游DNS服务器

2. **服务注册API端点** (任务2.5-2.7)
   - 实现服务注册API
   - 实现服务注销API
   - 实现服务心跳API

3. **动态服务发现** (任务2.8-2.9)
   - 实现etcd Watcher监听服务变化
   - 实现DNS服务器动态更新

## 依赖库

当前项目使用的主要外部依赖:

- `go.uber.org/zap` - 日志库
- `github.com/spf13/viper` - 配置管理
- `go.etcd.io/etcd/client/v3` - etcd客户端
- `github.com/labstack/echo/v4` - Web框架
- `github.com/miekg/dns` - DNS服务器库
- `github.com/stretchr/testify` - 测试辅助库

## 设计原则

项目严格遵循接口驱动开发原则:
- 所有服务和存储操作都先定义接口
- 然后再实现具体实现类
- 这有利于单元测试和模拟依赖

## 测试覆盖

已完成的模块都有对应的测试:
- 日志模块: 单元测试
- 配置模块: 单元测试
- etcd客户端: 集成测试
- API处理器: 单元测试
- DNS服务器: 单元测试 