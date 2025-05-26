# DNS 服务与服务发现系统开发规划文档

## 1. 引言 (Introduction)

### 1.1. 项目背景
随着微服务架构的普及，服务注册与发现成为了关键组件。传统的 DNS 解析方式在动态变化的服务实例面前略显不足。本项目旨在开发一个结合 etcd 的 DNS 服务，实现服务的自动注册与发现，并通过一个 React 前端提供管理界面。

### 1.2. 项目目标
*   构建一个高性能、高可用的 DNS 服务，能够从 etcd 中动态获取服务信息。
*   实现服务的自动注册机制，当新服务启动或停止时，自动更新 etcd 中的记录。
*   DNS 服务应能遵循上游 DNS 服务器的解析规则，对于 etcd 中未找到的域名，转发至上游 DNS 处理。
*   提供一个用户友好的 React 前端管理界面，用于查看服务状态、配置 DNS 记录等。
*   确保系统的可扩展性、可靠性和安全性。

### 1.3. 项目范围
*   **后端服务 (Go)**：
    *   DNS 服务器核心逻辑实现。
    *   与 etcd 的集成，包括服务注册（通过自身 API 代理）、监听服务变化、服务发现。
    *   向上游 DNS 服务器的查询转发。
    *   提供 RESTful API 供前端调用以及供服务实例进行注册/注销。
*   **前端界面 (React)**：
    *   展示已注册服务列表及其状态。
    *   允许用户查询和管理部分 DNS 配置（例如，自定义内部域名解析）。
    *   系统状态监控仪表盘。
*   **服务注册客户端 (可选)**：提供轻量级客户端库或规范，方便应用服务接入。

## 2. 需求分析 (Requirements Analysis)

### 2.1. 功能性需求
*   **FR1**: DNS 服务能正确解析在 etcd 中注册的域名。
*   **FR2**: DNS 服务能将未在 etcd 中找到的域名查询请求转发到配置的上游 DNS 服务器。
*   **FR3**: 服务实例启动时，能够通过调用 Go DNS 服务提供的 API，自动向 etcd 注册其服务名、IP 地址、端口等信息。
*   **FR4**: 服务实例关闭或异常时，能够通过调用 Go DNS 服务提供的 API，从 etcd 中自动注销。
*   **FR5**: React 前端能展示当前所有已注册的服务及其健康状态。
*   **FR6**: React 前端允许管理员进行必要的 DNS 配置（如上游 DNS 地址）。
*   **FR7**: API 接口提供服务列表查询、服务详情查询功能。
*   **FR8**: 支持常见的 DNS 记录类型 (A, AAAA, SRV, CNAME)。

### 2.2. 非功能性需求
*   **NFR1 (性能)**: DNS 查询响应时间应尽可能低，例如 P99 小于 50ms。
*   **NFR2 (高可用性)**: DNS 服务和 etcd 集群应具备高可用性，避免单点故障。
*   **NFR3 (可扩展性)**: 系统应能水平扩展以应对增长的服务数量和查询负载。
*   **NFR4 (安全性)**: API 接口需要认证和授权机制；敏感配置信息需要妥善管理。
*   **NFR5 (可维护性)**: 代码结构清晰，模块化设计，有良好的文档和注释。
*   **NFR6 (易用性)**: 前端界面操作直观，易于上手。

## 3. 系统架构 (System Architecture)

```
+---------------------+      +---------------------+      +---------------------+
| Service Instance A  |----->|                     |<-----| Service Instance B  |
| (Calls Reg. API)    |      |   Go DNS Service    |      | (Calls Reg. API)    |
+---------------------+      |  (Provides Reg. API |      +---------------------+
        ^                    |   & Talks to etcd)  |          ^
        | (DNS Query)        +----------|----------+          | (DNS Query)
        |                                | (Internal etcd ops) |
        v                                v                     v
+---------------------+      +---------------------+      +---------------------+
| DNS Client (App/OS) |----->| (Handles DNS Query) |<-----| React Admin Console |
+---------------------+      |                     |      | (Uses Mgmt API)     |
                             |  (Upstream Fallback)|      +---------------------+
                             +----------|----------+
                                        |
                                        v
                             +---------------------+
                             |   Upstream DNS      |
                             +---------------------+
                                        ^
                                        | (Internal etcd ops)
                               +--------|--------+
                               |  etcd Cluster   |
                               | (Internal Store)|
                               +-----------------+
```

*   **服务实例 (Service Instances)**: 运行业务应用的实例。启动时通过调用 **Go DNS 服务提供的服务注册 API** 来注册自身信息（如服务名、IP、端口）。关闭或出现故障时，也通过相应的 API 通知 Go DNS 服务进行注销。
*   **etcd 集群 (etcd Cluster)**: 作为 **内部核心数据存储**，用于持久化服务注册信息和可能的 DNS 配置。**它不直接对外暴露**，所有对其的读写操作（包括服务注册、发现、配置变更）都由 **Go DNS 服务代理完成**。
*   **Go DNS 服务 (Go DNS Service)**: 系统的核心组件，承担双重职责：
    *   **DNS 解析服务**: 监听标准的 DNS 查询请求 (UDP/53, TCP/53)。根据 etcd 中存储的服务信息进行解析。若内部无记录，则向上游 DNS 转发查询。
    *   **服务注册与管理 API**: 提供 RESTful API。一部分 API 供服务实例进行注册、注销和发送心跳；另一部分 API 供 React 管理控制台进行系统配置和状态监控。Go DNS 服务内部的 `etcd_client` 模块负责将通过 API 收到的注册信息同步到 etcd 集群，并监听 etcd 的变化以更新 DNS 解析数据。
*   **React 管理控制台 (React Admin Console)**:
    *   通过 API 与 Go DNS 服务交互。
    *   展示服务状态，允许用户配置。
*   **DNS 客户端 (DNS Client)**: 应用程序或操作系统，向 Go DNS 服务发起域名解析请求。
*   **上游 DNS (Upstream DNS)**: 公共或内部的权威 DNS 服务器，用于解析外部域名。

## 4. 模块设计 (Module Design)

### 4.1. 后端 (Go)

#### 4.1.1. DNS 服务器模块 (`dns_server`)
*   **职责**: 监听和处理 DNS 请求。
*   **主要组件**:
    *   `Server`: 启动和管理 DNS 监听 (UDP/TCP)。
    *   `Handler`: 解析 DNS 请求报文，根据域名类型分发处理。
    *   `Resolver`: 核心解析逻辑，包含 etcd 查询和上游查询。
*   **关键技术**: `net/dns` 包，goroutines 处理并发请求。

#### 4.1.2. etcd 交互模块 (`etcd_client`)
*   **职责**: 作为 Go DNS 服务内部模块，封装与 etcd 集群的所有交互。处理来自 API 层的服务注册/注销请求并将其持久化到 etcd，监听 etcd 中服务节点的变化以实时更新 DNS 服务的内部解析缓存，并从中发现服务信息以响应 DNS 查询。
*   **主要组件**:
    *   `RegistryPersistence`: 负责将服务注册、注销信息写入 etcd，处理租约和心跳的更新逻辑。
    *   `DiscoveryProvider`: 从 etcd 读取服务信息，供 DNS Resolver 使用。
    *   `Watcher`: 监听 etcd 中服务节点的变化，实时通知相关模块（如 DNS Resolver 的缓存）。
*   **关键技术**: etcd V3 client (`go.etcd.io/etcd/client/v3`)。

#### 4.1.3. API 接口模块 (`api_handler`)
*   **职责**: 提供 RESTful API。一部分供 React 前端管理控制台调用（如查询服务列表、获取配置），另一部分供服务实例调用以完成服务注册、注销和心跳维持。
*   **主要组件**:
    *   `Router`: 定义 API 路由 (e.g., `/admin/services`, `/admin/config`, `/services/register`, `/services/deregister`, `/services/heartbeat`)。
    *   `AuthMiddleware`: API 认证与授权 (可能对管理 API 和服务注册 API 采用不同策略)。
    *   `ServiceController`: 处理服务相关的管理 API 请求 (如列表查询)。
    *   `RegistryController`: 处理服务实例的注册、注销、心跳 API 请求，内部调用 `etcd_client` 模块。
    *   `ConfigController`: 处理配置相关的 API 请求。
*   **关键技术**: Gin 或 Echo Web 框架。

#### 4.1.4. 配置模块 (`config`)
*   **职责**: 加载和管理系统配置 (e.g., etcd 地址, 上游 DNS, API 端口)。
*   **主要组件**:
    *   `Loader`: 从文件或环境变量加载配置。
    *   `Store`: 存储当前配置。
*   **关键技术**: Viper 或原生 `os` 和 `json/yaml` 包。

### 4.2. 前端 (React)

#### 4.2.1. 服务管理模块 (`ServiceManagement`)
*   **职责**: 展示服务列表、服务详情、服务状态。
*   **组件**: `ServiceTable`, `ServiceDetailView`, `StatusIndicator`.
*   **状态管理**: Redux Toolkit 或 Zustand。

#### 4.2.2. DNS 配置模块 (`DNSConfig`)
*   **职责**: 允许用户配置上游 DNS，查看或修改少量自定义解析规则。
*   **组件**: `ConfigForm`, `RuleList`.

#### 4.2.3. API 客户端模块 (`apiClient`)
*   **职责**: 封装对后端 API 的调用。
*   **技术**: Axios 或 Fetch API。

#### 4.2.4. 仪表盘模块 (`Dashboard`)
*   **职责**: 展示系统关键指标，如查询 QPS、etcd 连接状态等。
*   **组件**: `StatCard`, `ChartComponent`.

## 5. 数据模型 (Data Model - Stored in etcd)

### 5.1. 服务注册信息
*   **Key**: `/services/{namespace}/{service_name}/{instance_id}`
*   **Value (JSON)**:
    ```json
    {
      "service_name": "my-app",
      "instance_id": "uuid-xxxx-xxxx",
      "ip_address": "10.0.1.10",
      "port": 8080,
      "metadata": { // 可选元数据
        "version": "v1.2.3",
        "region": "us-west-1"
      },
      "last_heartbeat": "timestamp", // 用于健康检查
      "ttl": 30 // 租约 TTL (秒)
    }
    ```
    *   **域名生成规则**: `{service_name}.{namespace}.svc.cluster.local` (可配置)。SRV 记录将包含 IP 和 Port。

### 5.2. 自定义 DNS 记录 (可选，如果支持管理界面配置)
*   **Key**: `/dns/records/{domain_name}/{record_type}`
*   **Value (JSON)**:
    ```json
    {
      "type": "A", // A, AAAA, CNAME, SRV, TXT
      "value": "192.168.1.100", // 对于 A 记录是 IP, CNAME 是目标域名等
      "ttl": 300
    }
    ```

## 6. 技术选型 (Technology Stack)

*   **后端**:
    *   **语言**: Go (latest stable version)
    *   **DNS 库**: `miekg/dns` (比 `net/dns` 更灵活和完整)
    *   **etcd 客户端**: `go.etcd.io/etcd/client/v3`
    *   **Web 框架 (API)**: Gin Gonic (`github.com/gin-gonic/gin`) 或 Echo (`github.com/labstack/echo`)
    *   **配置管理**: Viper (`github.com/spf13/viper`)
    *   **日志**: Logrus (`github.com/sirupsen/logrus`) 或 Zap (`go.uber.org/zap`)
    *   **依赖管理**: Go Modules
*   **前端**:
    *   **框架**: React (latest stable version)
    *   **脚手架**: Create React App or Vite
    *   **状态管理**: Redux Toolkit or Zustand
    *   **HTTP 客户端**: Axios
    *   **UI 组件库**: Material-UI, Ant Design, or Tailwind CSS
*   **服务注册与发现**: etcd (latest stable v3.x)
*   **版本控制**: Git, GitHub/GitLab
*   **容器化**: Docker, Docker Compose (for local dev/testing)
*   **CI/CD**: GitHub Actions, GitLab CI, or Jenkins

## 7. 开发计划 (Development Plan)

| 阶段   | 任务                                     | 负责人 | 预计时间 | 状态     |
| ------ | ---------------------------------------- | ------ | -------- | -------- |
| **M1: 核心 DNS 服务与 etcd 集成 (4 周)** |                                          |        |          |          |
|        | 1. 项目初始化、Go Modules、基本目录结构    | Dev    | 2 天     | To Do    |
|        | 2. etcd 客户端封装 (连接、CRUD、Watch)    | Dev    | 5 天     | To Do    |
|        | 3. DNS 服务器基本框架 (`miekg/dns`)       | Dev    | 3 天     | To Do    |
|        | 4. 实现从 etcd 读取服务信息并解析 A/SRV 记录 | Dev    | 7 天     | To Do    |
|        | 5. 实现向上游 DNS 转发逻辑               | Dev    | 3 天     | To Do    |
| **M2: 服务注册 API 与 etcd 持久化 (3 周)** |                                          |        |          |          |
|        | 1. 设计服务注册 API 端点及请求/响应模型    | Dev    | 3 天     | To Do    |
|        | 2. 实现服务注册、注销、心跳 API 处理器 (调用 etcd_client) | Dev    | 7 天     | To Do    |
|        | 3. `etcd_client` 实现服务信息到 etcd 的持久化逻辑 (含租约) | Dev    | 5 天     | To Do    |
|        | 4. API: (管理端) 查询服务列表、服务详情    | Dev    | 3 天     | To Do    |
| **M3: React 前端基础与服务展示 (4 周)** |                                          |        |          |          |
|        | 1. React 项目初始化 (CRA/Vite)          | Dev    | 2 天     | To Do    |
|        | 2. API Client 封装 (Axios)               | Dev    | 3 天     | To Do    |
|        | 3. 基础布局、路由、导航组件                | Dev    | 5 天     | To Do    |
|        | 4. 服务列表展示页面 (调用 API)             | Dev    | 5 天     | To Do    |
|        | 5. 服务详情展示页面                      | Dev    | 5 天     | To Do    |
| **M4: 前端配置与高级功能 (3 周)**     |                                          |        |          |          |
|        | 1. DNS 配置管理页面 (上游 DNS)           | Dev    | 5 天     | To Do    |
|        | 2. (可选) 自定义 DNS 记录管理            | Dev    | 5 天     | To Do    |
|        | 3. 仪表盘页面初步实现                    | Dev    | 5 天     | To Do    |
| **M5: 测试、文档与部署准备 (2 周)**   |                                          |        |          |          |
|        | 1. 单元测试 (Go & React)                 | Dev    | 5 天     | To Do    |
|        | 2. 集成测试 (DNS 服务与 etcd)            | Dev    | 3 天     | To Do    |
|        | 3. 编写用户文档和部署文档                | Dev    | 2 天     | To Do    |
|        | 4. Dockerfile 和 Docker Compose 配置     | Dev    | 2 天     | To Do    |
| **M6: 部署与优化 (持续)**           |                                          |        |          |          |

## 8. 测试计划 (Testing Plan)

### 8.1. 单元测试 (Unit Tests)
*   **Go**: 使用 Go 原生的 `testing` 包。测试各个模块的独立功能，如 DNS 解析逻辑、etcd 客户端操作、API 处理器。使用 `mock` 模拟外部依赖 (如 etcd client, 上游 DNS)。
*   **React**: 使用 Jest 和 React Testing Library。测试组件的渲染、用户交互和状态变化。

### 8.2. 集成测试 (Integration Tests)
*   测试 Go DNS 服务与 etcd 的集成：服务注册后能否被正确解析。
*   测试 DNS 查询转发到上游 DNS 的流程。
*   测试 API 接口与后端逻辑的连通性。

### 8.3. 端到端测试 (End-to-End Tests)
*   模拟用户场景，从服务注册、前端配置、DNS 查询到最终解析结果的完整流程。
*   可以使用 Cypress 或 Playwright 等工具进行前端 E2E 测试。

### 8.4. 性能测试
*   使用 `dnsperf` 或 `queryperf` 等工具测试 DNS 服务的 QPS 和延迟。
*   对 API 接口进行压力测试。

## 9. 部署方案 (Deployment Plan)

### 9.1. 开发环境
*   本地运行 etcd 实例 (Docker)。
*   本地运行 Go DNS 服务。
*   本地运行 React 开发服务器。
*   使用 Docker Compose 统一管理本地开发环境。

### 9.2. 测试环境
*   与生产环境相似的独立环境。
*   部署 etcd 集群。
*   部署多个 Go DNS 服务实例。
*   部署 React 前端。
*   自动化测试流程。

### 9.3. 生产环境
*   **etcd**: 部署高可用的 etcd 集群 (至少3个节点)。
*   **Go DNS Service**:
    *   以容器方式 (Docker) 部署。
    *   使用 Kubernetes 或其他容器编排平台进行管理，实现自动扩缩容和故障恢复。
    *   通过 Load Balancer 将 DNS 请求分发到多个实例。
*   **React Admin Console**:
    *   构建静态文件，通过 Nginx 或 CDN 提供服务。
    *   API 请求指向后端 Load Balancer。
*   **配置管理**: 使用 ConfigMap (Kubernetes) 或专门的配置中心。
*   **日志与监控**: 集成 Prometheus, Grafana, ELK Stack 等。

## 10. 风险评估与应对 (Risk Assessment and Mitigation)

| 风险点                     | 可能性 | 影响程度 | 应对措施                                                                 |
| -------------------------- | ------ | -------- | ------------------------------------------------------------------------ |
| etcd 集群不稳定/故障      | 中     | 高       | 部署高可用 etcd 集群；DNS 服务端实现对 etcd 连接的重试和容错机制；监控 etcd 健康状态。 |
| DNS 服务性能瓶颈         | 中     | 中       | 优化代码；使用连接池；水平扩展 DNS 服务实例；使用高效的 DNS 库。                    |
| 上游 DNS 服务不可用      | 低     | 中       | 配置多个上游 DNS 服务器；实现缓存机制；监控上游 DNS 状态。                         |
| 安全漏洞 (API, etcd)     | 中     | 高       | API 认证授权；限制 etcd 访问权限；定期安全审计；依赖库版本更新。                   |
| 服务注册/注销逻辑错误    | 中     | 中       | 充分的单元测试和集成测试；设计合理的租约和心跳机制。                               |
| 前后端接口不兼容         | 中     | 低       | 定义清晰的 API 规范 (OpenAPI/Swagger)；前后端协同开发和测试。                      |
| 技术选型不当导致后期维护困难 | 低     | 中       | 充分调研，选择成熟稳定的技术栈；遵循社区最佳实践。                               |

## 11. 未来展望

*   支持更多 DNS 记录类型。
*   更细粒度的权限控制。
*   与服务网格 (Service Mesh) 如 Istio, Linkerd 集成。
*   提供更丰富的监控指标和告警。
*   基于 Webhook 的事件通知。 