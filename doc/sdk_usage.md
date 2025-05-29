# Kong Discovery SDK 使用指南

本文档提供了Kong Discovery SDK的详细使用说明，包括安装、配置、示例和故障排除。

## 目录

- [安装](#安装)
- [组件介绍](#组件介绍)
- [服务注册](#服务注册)
- [服务发现](#服务发现)
- [HTTP客户端](#http客户端)
- [故障排除](#故障排除)

## 安装

通过Go模块安装SDK：

```bash
go get github.com/hewenyu/kong-discovery
```

## 组件介绍

Kong Discovery SDK主要包含以下组件：

1. **KongDiscoveryClient**: 负责服务注册、注销和心跳
2. **DNSDiscovery**: 负责通过DNS进行服务发现
3. **ServiceDiscoveryHTTPClient**: 基于服务发现的HTTP客户端示例

## 服务注册

服务注册是将服务实例信息注册到Kong Discovery系统的过程。

### 基本用法

```go
// 创建客户端 (可以指定服务器地址)
client := sdk.NewClient("http://kong-discovery-server:8081")

// 或使用默认地址 (localhost:8081)
client := sdk.NewDefaultClient("")

// 准备服务实例信息
serviceInstance := &sdk.ServiceInstance{
    ServiceName: "my-service",       // 服务名称
    InstanceID:  "instance-001",     // 实例ID，确保唯一性
    IPAddress:   "192.168.1.100",    // 服务IP地址
    Port:        8080,               // 服务端口
    TTL:         60,                 // 租约TTL（秒）
    Metadata: map[string]string{     // 可选元数据
        "version": "1.0.0",
        "region":  "cn-north",
    },
}

// 注册服务
ctx := context.Background()
response, err := client.Register(ctx, serviceInstance)
if err != nil {
    log.Fatalf("服务注册失败: %v", err)
}
fmt.Printf("服务注册成功: %+v\n", response)

// 启动心跳循环 (间隔30秒，TTL 60秒)
client.StartHeartbeatLoop(ctx, serviceInstance.ServiceName, serviceInstance.InstanceID, 30*time.Second, 60)

// 当服务需要下线时，注销服务
deregisterResp, err := client.Deregister(ctx, serviceInstance.ServiceName, serviceInstance.InstanceID)
```

### 获取本机IP地址

在实际应用中，你可能需要自动获取本机IP地址而不是硬编码：

```go
func getIPAddress() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        log.Fatalf("获取IP地址失败: %v", err)
    }
    for _, addr := range addrs {
        if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String()
            }
        }
    }
    return "127.0.0.1" // 如果找不到，返回本地回环地址
}

// 然后在服务实例中使用
serviceInstance.IPAddress = getIPAddress()
```

### 优雅退出

确保在应用程序关闭时正确注销服务：

```go
// 监听系统信号
signalChan := make(chan os.Signal, 1)
signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
go func() {
    <-signalChan
    fmt.Println("正在关闭应用...")
    
    // 注销服务
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    client.Deregister(ctx, serviceName, instanceID)
    
    os.Exit(0)
}()
```

## 服务发现

服务发现是通过DNS查询来发现已注册服务的过程。

### 基本用法

```go
// 创建DNS服务发现客户端
discovery := sdk.NewDNSDiscovery("127.0.0.1:6553", 60*time.Second)

// 解析主机地址 (A记录)
host, err := discovery.ResolveHost(ctx, "my-service")
if err == nil {
    fmt.Printf("服务地址: %s\n", host)
}

// 解析SRV记录
srv, err := discovery.ResolveSRV(ctx, "my-service")
if err == nil {
    fmt.Printf("服务SRV记录: %s:%d\n", srv.Target, srv.Port)
}

// 综合解析服务 (优先使用SRV记录，失败则使用A记录)
serviceAddr, err := discovery.ResolveService(ctx, "my-service")
if err == nil {
    fmt.Printf("服务地址: %s\n", serviceAddr)
}
```

### 域名格式

服务发现支持两种格式的域名：

1. **简单名称**: 如`my-service`，会自动转换为`my-service.service.discovery`进行A记录查询，或`_my-service._tcp.service.discovery`进行SRV查询
2. **完整域名**: 如`my-service.example.com`，将直接使用此域名进行查询

## HTTP客户端

SDK提供了一个基于服务发现的HTTP客户端示例。

### 基本用法

```go
// 创建HTTP客户端
client := &ServiceDiscoveryHTTPClient{
    discovery: sdk.NewDNSDiscovery("127.0.0.1:6553", 60*time.Second),
    client: &http.Client{Timeout: 10*time.Second},
}

// 发送GET请求到服务的/health端点
resp, err := client.Get(ctx, "my-service", "/health")
if err == nil {
    fmt.Printf("响应: %s\n", string(resp))
}
```

### 自定义HTTP客户端

```go
// 基于服务发现的HTTP客户端
type ServiceDiscoveryHTTPClient struct {
    discovery *sdk.DNSDiscovery
    client    *http.Client
}

// 发送HTTP请求到指定服务
func (c *ServiceDiscoveryHTTPClient) Request(ctx context.Context, serviceName, path, method string) ([]byte, error) {
    // 使用服务发现解析服务地址
    serviceAddr, err := c.discovery.ResolveService(ctx, serviceName)
    if err != nil {
        return nil, fmt.Errorf("解析服务地址失败: %w", err)
    }

    // 构建完整URL
    url := fmt.Sprintf("http://%s%s", serviceAddr, path)
    
    // 创建HTTP请求
    req, err := http.NewRequestWithContext(ctx, method, url, nil)
    if err != nil {
        return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
    }

    // 发送HTTP请求
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
    }
    defer resp.Body.Close()

    // 读取响应体
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("读取响应体失败: %w", err)
    }

    return body, nil
}
```

## 故障排除

### 服务注册问题

1. **无法连接到Kong Discovery服务**

   **症状**: 出现错误如 `服务注册失败: 发送HTTP请求失败: dial tcp: lookup kong-discovery-server`
   
   **解决方法**:
   - 确保Kong Discovery服务正在运行
   - 检查服务地址是否正确
   - 检查网络连接是否正常

   ```go
   // 检查Kong Discovery服务连接
   _, err := http.Get("http://kong-discovery-server:8081/health")
   if err != nil {
       fmt.Printf("Kong Discovery服务不可访问: %v\n", err)
   }
   ```

2. **注册成功但心跳失败**

   **症状**: 注册成功但心跳日志中出现错误
   
   **解决方法**:
   - 确认心跳间隔不要过短（建议30秒或更长）
   - 确认TTL设置合理（至少是心跳间隔的2倍）
   - 确认服务名和实例ID与注册时一致

### 服务发现问题

1. **DNS服务器连接失败**

   **症状**: 错误信息如 `解析服务[my-service.service.discovery]失败: lookup my-service.service.discovery on 192.168.1.1:53: no such host`
   
   **解决方法**:
   - 确保Kong Discovery的DNS服务正在运行并监听正确的端口
   - 确认指定的DNS服务器地址和端口正确
   - 检查是否有防火墙阻止UDP流量

   ```go
   // 检查DNS服务器连接
   conn, err := net.DialTimeout("udp", "127.0.0.1:6553", 2*time.Second)
   if err != nil {
       fmt.Printf("无法连接到DNS服务器: %v\n", err)
   } else {
       conn.Close()
       fmt.Println("DNS服务器连接成功")
   }
   ```

2. **找不到服务**

   **症状**: 错误信息如 `未找到服务[my-service.service.discovery]的地址`
   
   **解决方法**:
   - 确认服务已经成功注册
   - 检查服务名称是否正确
   - 确认Kong Discovery的etcd和DNS服务之间通信正常

3. **解析器不使用指定的DNS服务器**

   **症状**: DNS查询被发送到系统默认DNS服务器而不是Kong Discovery DNS服务器
   
   **解决方法**:
   - 确保使用了SDK中的`NewDNSDiscovery`函数创建解析器
   - 检查DNS服务器地址是否正确
   - 使用`dig`或`nslookup`工具测试DNS服务器是否正常工作

   ```bash
   # 测试DNS服务器
   dig @127.0.0.1 -p 6553 my-service.service.discovery
   ```

### HTTP客户端问题

1. **无法连接到服务**

   **症状**: 错误信息如 `发送HTTP请求失败: dial tcp: connect: connection refused`
   
   **解决方法**:
   - 确认服务发现是否正确解析了地址
   - 确认目标服务正在运行并监听指定端口
   - 检查网络连接是否正常

2. **服务响应错误**

   **症状**: 收到非200响应码
   
   **解决方法**:
   - 检查请求路径是否正确
   - 确认服务健康状态
   - 查看服务日志以获取更多信息

## 日志和调试

在排查问题时，增加日志输出可以帮助定位问题：

```go
// 在DNS解析前打印信息
fmt.Printf("正在解析服务: %s 使用DNS服务器: %s\n", serviceName, dnsServer)

// 捕获和记录错误
if err != nil {
    fmt.Printf("操作失败，详细错误: %v\n", err)
}
```

## 性能优化

1. **调整缓存TTL**: 根据服务变更频率设置合适的缓存TTL，可以减少DNS查询次数
2. **使用长连接**: HTTP客户端默认使用连接池，确保不要频繁创建新的客户端实例
3. **合理设置超时**: 设置合适的HTTP和DNS请求超时时间，避免长时间等待 