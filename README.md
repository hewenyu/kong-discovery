# Kong Discovery

Kong Discovery是一个为Kong网关提供服务发现功能的系统。它结合了DNS服务与etcd，实现了服务的自动注册与发现机制。

## 功能特点

- 基于DNS的服务发现，支持A记录和SRV记录
- 与etcd集成，提供高可用的服务注册存储
- 支持服务的动态注册、注销和心跳检测
- 支持配置多个上游DNS服务器，实现负载均衡和故障转移
- 提供Web管理界面，方便服务和DNS配置管理
- 监听多个端口，管理API和服务注册API分离，增强安全性

## 上游DNS配置

Kong Discovery支持配置多个上游DNS服务器，以实现：

1. **负载均衡**：采用轮询方式在多个上游DNS服务器之间分发查询请求
2. **故障转移**：当一个上游DNS服务器无响应时，自动尝试下一个服务器

配置方式：

```yaml
dns:
  upstream_dns:
    - "8.8.8.8:53"   # Google DNS
    - "1.1.1.1:53"   # Cloudflare DNS
    - "114.114.114.114:53"  # 国内DNS
```

可以通过Web管理界面或API接口动态修改上游DNS配置。

## 安装与使用

详细的安装和使用说明请参考[文档](doc/project_structure.md)。

## 变更日志

查看[变更日志](doc/changelog.md)了解项目的最新变更。
