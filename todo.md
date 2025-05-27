# 项目任务清单 (TODO)

**开发原则：**

*   严格按照任务列表顺序进行开发。
*   每个任务完成后，必须为其编写单元测试和必要的集成测试。
*   只有当前任务及其测试全部通过后，才能开始下一个任务。
*   标记 `- [x]` 表示任务已完成并通过测试。
*   标记 `- [ ]` 表示任务待办。


* **必须使用接口驱动开发**：
  - 所有服务和存储操作都必须先定义接口
  - 然后再实现具体的实现类
  - 这样有利于单元测试
---

## 第一阶段：Go 后端 - 基础环境与核心 DNS 功能

### Go 项目初始化与基础配置
- [x] **1.1.** 初始化 Go Modules (`go mod init`)，创建项目基本目录结构 (e.g., `/cmd`, `/internal`, `/pkg`, `/configs`, `/api`)。
    *   *测试*：确认目录结构正确，`go build` 空 `main` 函数能通过。
- [x] **1.2.** 集成 Zap 日志库：进行基础配置，提供一个全局可用的 logger 实例。
    *   *测试*：编写简单的日志输出测试，验证日志格式和级别。
- [x] **1.3.** 配置管理模块 (`internal/config`)：实现从文件 (e.g., `config.yaml`) 或环境变量加载基础配置（如 etcd 地址、DNS 监听端口、管理 API 端口、服务注册 API 端口）。
    *   *测试*：单元测试配置加载逻辑，覆盖不同来源和缺失配置的情况。
- [x] **1.4.** etcd 客户端模块 (`internal/etcdclient`)：实现连接 etcd、检查连接状态 (Ping) 的基础功能。
    *   *测试*：单元测试连接和 Ping etcd 的功能 (可能需要 mock etcd client 或连接到本地测试 etcd)。

### Echo 框架与 API 基础
- [ ] **1.5.** API 处理器模块 (`internal/apihandler`)：使用 Echo 框架搭建基础 HTTP 服务，能够启动并监听指定的**管理 API 端口**。
    *   *测试*：启动服务后，通过 `curl` 或 HTTP 客户端访问一个健康的检查端点 (e.g., `/health`)，确认服务能正常响应。
- [ ] **1.6.** API 处理器模块 (`internal/apihandler`)：配置 Echo 服务以监听指定的**服务注册 API 端口** (可以与管理 API 在同一个 Echo 实例中通过不同 Server 配置，或两个独立实例)。
    *   *测试*：启动服务后，确认两个端口都能独立响应健康的检查端点。

### 核心 DNS 服务
- [ ] **1.7.** DNS 服务器模块 (`internal/dnsserver`)：使用 `miekg/dns` 库搭建基础 DNS 服务器，监听配置的 DNS 端口 (UDP/53)。
    *   *测试*：实现一个硬编码的 DNS 记录响应 (e.g., `test.local A 1.2.3.4`)，使用 `dig` 或 `nslookup` 查询并验证结果。
- [ ] **1.8.** `etcdclient` 模块：实现从 etcd 读取一个指定 key 的 PoC (Proof of Concept) 功能 (用于后续服务发现)。
    *   *测试*：单元测试从 etcd 读取特定 key-value 的逻辑。
- [ ] **1.9.** `dnsserver` 模块：集成 `etcdclient`，修改 DNS 处理逻辑，尝试从 etcd 读取服务信息 (基于查询的域名构造 etcd key) 并响应 A/SRV 记录。
    *   *测试*：在 etcd 中预设服务数据，通过 `dig` 查询对应域名，验证 DNS 解析结果是否与 etcd 数据一致。
- [ ] **1.10.** `dnsserver` 模块：实现向上游 DNS 服务器转发未知域名的查询逻辑。
    *   *测试*：配置上游 DNS，查询一个外部域名，验证是否能正确从上游获取结果。

## 第二阶段：Go 后端 - 服务注册 API 与动态发现

### 服务注册逻辑 (etcd 持久化)
- [ ] **2.1.** `etcdclient` 模块：设计服务在 etcd 中存储的 key 结构和 value (JSON) 格式。
    *   *测试*：文档评审，确保设计合理。
- [ ] **2.2.** `etcdclient` 模块：实现将服务信息写入 etcd 的核心逻辑，包括使用租约 (Lease) 和 TTL。
    *   *测试*：单元测试服务信息写入、租约创建和 TTL 设置。
- [ ] **2.3.** `etcdclient` 模块：实现从 etcd 删除服务信息的逻辑。
    *   *测试*：单元测试服务信息删除。
- [ ] **2.4.** `etcdclient` 模块：实现刷新服务租约（心跳）的逻辑。
    *   *测试*：单元测试租约刷新。

### 服务注册 API 端点 (Echo)
- [ ] **2.5.** `apihandler` 模块：在**服务注册 API 端口**上，设计并实现服务注册 API 端点 (`POST /services/register`)。接收服务注册信息，调用 `etcdclient` 写入 etcd。
    *   *测试*：集成测试，通过 API 注册服务，然后检查 etcd 中是否出现对应数据，并验证 DNS 解析（复用任务 1.9 的测试逻辑）。
- [ ] **2.6.** `apihandler` 模块：在**服务注册 API 端口**上，实现服务注销 API 端点 (`DELETE /services/{serviceName}/{instanceId}`)。调用 `etcdclient` 从 etcd 删除。
    *   *测试*：集成测试，注册服务 -> API 注销服务 -> 检查 etcd 数据 -> 验证 DNS 解析不再返回该实例。
- [ ] **2.7.** `apihandler` 模块：在**服务注册 API 端口**上，实现服务心跳 API 端点 (`PUT /services/heartbeat/{serviceName}/{instanceId}`)。调用 `etcdclient` 刷新租约。
    *   *测试*：集成测试，注册服务 -> 等待一段时间（接近 TTL 但未超时） -> 发送心跳 -> 验证 etcd 中租约已续期。

### 动态服务发现与 DNS 更新
- [ ] **2.8.** `etcdclient` 模块：实现 Watcher 逻辑，监听 etcd 中服务路径（e.g., `/services/`) 下的 key 变化。
    *   *测试*：单元测试 Watcher 能够正确接收到 etcd 的 create, update, delete 事件。
- [ ] **2.9.** `dnsserver` 模块：集成 `etcdclient` 的 Watcher。当服务发生变化时，动态更新 DNS 服务器内部的解析数据/缓存（或标记需要重新查询）。
    *   *测试*：集成测试，动态在 etcd 中添加/删除服务，通过 `dig` 查询，验证 DNS 解析结果能实时反映变化。

### 管理 API 端点 (Echo)
- [ ] **2.10.** `apihandler` 模块：在**管理 API 端口**上，设计并实现获取服务列表 API 端点 (`GET /admin/services`)。从 `etcdclient` 读取当前所有已注册服务。
    *   *测试*：集成测试，注册若干服务 -> 调用 API -> 验证返回的服务列表与 etcd 中一致。
- [ ] **2.11.** `apihandler` 模块：在**管理 API 端口**上，实现获取服务详情 API 端点 (`GET /admin/services/{serviceName}/{instanceId}`)。
    *   *测试*：集成测试，调用 API 获取特定服务详情。

## 第三阶段：React 前端 - 基础搭建与服务展示

- [ ] **3.1.** React 项目初始化：使用 Create React App 或 Vite 初始化前端项目，配置基本目录结构。
    *   *测试*：项目能成功启动，显示默认页面。
- [ ] **3.2.** API Client 封装：创建 Axios (或 Fetch) 实例，封装基础的 API 调用逻辑 (连接到后端管理 API 端口)。
    *   *测试*：尝试调用后端的 `/health` 或 `/admin/services` (空列表)，验证连接。
- [ ] **3.3.** 基础 UI 框架：实现应用的整体布局组件 (如 Header, Sidebar, Content Area) 和基本路由 (e.g., using React Router)。
    *   *测试*：手动检查页面布局和路由跳转是否正常。
- [ ] **3.4.** 服务列表页面：创建一个组件，调用 `/admin/services` API，获取并以表格形式展示服务列表。
    *   *测试*：后端预注册几个服务，前端页面能正确展示列表数据。
- [ ] **3.5.** 服务详情页面：创建组件，当用户在服务列表点击某服务时，路由到此页面，调用 `/admin/services/{serviceName}/{instanceId}` API，展示服务详细信息。
    *   *测试*：验证从列表页跳转到详情页，并正确显示服务数据。

## 第四阶段：高级功能与完善

- [ ] **4.1.** `apihandler` 模块：在**管理 API 端口**上，实现获取和更新上游 DNS 配置的 API 端点 (e.g., `GET /admin/config/upstream-dns`, `PUT /admin/config/upstream-dns`)。配置可存储于 etcd 或配置文件。
    *   *测试*：API 调用测试，验证配置的读取和更新。
- [ ] **4.2.** `dnsserver` 模块：修改 DNS 转发逻辑，使其从配置模块动态读取和使用上游 DNS 服务器地址。
    *   *测试*：通过 API 更新上游 DNS -> 查询外部域名 -> 验证 DNS 服务使用了新的上游。
- [ ] **4.3.** React 前端：创建 DNS 配置管理页面，允许用户查看和修改上游 DNS 服务器配置。
    *   *测试*：前端页面能正确显示和提交配置，后端配置相应更新。
- [ ] **4.4.** (可选) 实现自定义 DNS 记录管理 API 和前端页面 (如果需求明确)。
- [ ] **4.5.** (可选) 实现 React 仪表盘页面，展示一些基本的服务统计信息。

## 第五阶段：测试、文档与部署准备

- [ ] **5.1.** Go 后端：全面梳理并完善所有模块的单元测试，提高测试覆盖率。
- [ ] **5.2.** Go 后端：编写更全面的集成测试，覆盖主要用户场景（如服务完整生命周期、多种 DNS 查询类型）。
- [ ] **5.3.** React 前端：完善组件测试和单元测试。
- [ ] **5.4.** 端到端 (E2E) 测试：使用工具 (如 Cypress, Playwright，或简单的脚本) 测试一个完整的用户流程（服务注册 -> 前端展示 -> DNS 查询）。
- [ ] **5.5.** 文档：
    *   [ ] 完善 `README.md`：项目介绍、如何构建、运行、测试。
    *   [ ] API 文档：为 Go 后端 API 生成 OpenAPI/Swagger 文档 (Echo 支持此功能)。
    *   [ ] 简要的用户手册和部署指南。
- [ ] **5.6.** Docker化：
    *   [ ] 为 Go 后端服务创建 `Dockerfile`。
    *   [ ] 为 React 前端应用创建 `Dockerfile` (用于构建和提供静态文件)。
- [ ] **5.7.** Docker Compose：创建 `docker-compose.yml` 文件，用于在本地一键启动整个应用栈（etcd, Go DNS 服务, React 前端）。
    *   *测试*：`docker-compose up -d` 能成功启动所有服务，并且应用功能正常。

--- 