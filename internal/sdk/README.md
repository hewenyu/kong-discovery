# Kong Discovery SDK

Kong Discovery SDK是一个Go语言客户端库，用于与Kong Discovery服务交互，支持服务注册、服务注销、心跳维持和服务发现功能。

## 安装

```bash
go get github.com/hewenyu/kong-discovery
```

## 功能特性

* 服务注册：将服务实例注册到Kong Discovery系统
* 服务注销：从Kong Discovery系统中注销服务实例
* 服务心跳：定期发送心跳保持服务实例活跃
* 服务发现：通过DNS方式发现已注册的服务
* HTTP客户端：基于服务发现的HTTP客户端，支持自动解析服务地址

## 快速开始

### 服务注册与心跳

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hewenyu/kong-discovery/internal/sdk"
)

func main() {
	// 创建上下文，用于管理整个程序的生命周期
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建Kong Discovery客户端
	client := sdk.NewDefaultClient() // 默认连接到localhost:8081
	// 或者指定服务地址
	// client := sdk.NewClient("http://kong-discovery-api:8081")

	// 准备服务实例信息
	serviceInstance := &sdk.ServiceInstance{
		ServiceName: "my-service",
		InstanceID:  "instance-001",
		IPAddress:   "192.168.1.100",
		Port:        8080,
		TTL:         60, // 60秒租约
		Metadata: map[string]string{
			"version": "1.0.0",
			"region":  "cn-north",
		},
	}

	// 注册服务
	response, err := client.Register(ctx, serviceInstance)
	if err != nil {
		fmt.Printf("服务注册失败: %v\n", err)
		return
	}

	fmt.Printf("服务注册成功: %+v\n", response)

	// 启动心跳循环，在后台保持服务注册状态
	client.StartHeartbeatLoop(ctx, serviceInstance.ServiceName, serviceInstance.InstanceID, 30*time.Second, 60)

	// 监听退出信号
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	// 注销服务
	deregisterCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	client.Deregister(deregisterCtx, serviceInstance.ServiceName, serviceInstance.InstanceID)
}
```

### 服务发现

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hewenyu/kong-discovery/internal/sdk"
)

func main() {
	// 创建上下文
	ctx := context.Background()

	// 创建DNS服务发现客户端
	// 参数1: DNS服务器地址
	// 参数2: 缓存TTL
	discovery := sdk.NewDNSDiscovery("127.0.0.1:6553", 60*time.Second)

	// 解析服务地址 (A记录)
	host, err := discovery.ResolveHost(ctx, "my-service")
	if err != nil {
		fmt.Printf("解析服务地址失败: %v\n", err)
	} else {
		fmt.Printf("解析到服务地址: %s\n", host)
		// 输出类似: 192.168.1.100
	}

	// 解析SRV记录
	srv, err := discovery.ResolveSRV(ctx, "my-service")
	if err != nil {
		fmt.Printf("解析SRV记录失败: %v\n", err)
	} else {
		fmt.Printf("解析到SRV记录: Target=%s, Port=%d\n", srv.Target, srv.Port)
		// 输出类似: Target=instance-001.my-service.service.discovery., Port=8080
	}

	// 解析服务（综合A和SRV记录）
	service, err := discovery.ResolveService(ctx, "my-service")
	if err != nil {
		fmt.Printf("解析服务失败: %v\n", err)
	} else {
		fmt.Printf("解析到服务: %s\n", service)
		// 输出类似: 192.168.1.100:8080
	}
}
```

### 基于服务发现的HTTP客户端

以下示例展示如何使用服务发现来构建HTTP客户端，自动解析服务地址并发送请求:

```go
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hewenyu/kong-discovery/internal/sdk"
)

// 基于服务发现的HTTP客户端
type ServiceDiscoveryHTTPClient struct {
	discovery *sdk.DNSDiscovery
	client    *http.Client
}

// 创建基于服务发现的HTTP客户端
func NewServiceDiscoveryHTTPClient(dnsServer string, timeout time.Duration) *ServiceDiscoveryHTTPClient {
	return &ServiceDiscoveryHTTPClient{
		discovery: sdk.NewDNSDiscovery(dnsServer, 60*time.Second),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// 发送HTTP请求到指定服务
func (c *ServiceDiscoveryHTTPClient) Request(ctx context.Context, serviceName, path string, method string) ([]byte, error) {
	// 使用服务发现解析服务地址
	serviceAddr, err := c.discovery.ResolveService(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("解析服务地址失败: %w", err)
	}

	// 构建完整URL
	url := fmt.Sprintf("http://%s%s", serviceAddr, path)
	fmt.Printf("发送请求到: %s\n", url)

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

// 发送GET请求
func (c *ServiceDiscoveryHTTPClient) Get(ctx context.Context, serviceName, path string) ([]byte, error) {
	return c.Request(ctx, serviceName, path, http.MethodGet)
}

func main() {
	// 创建上下文
	ctx := context.Background()

	// 创建基于服务发现的HTTP客户端
	client := NewServiceDiscoveryHTTPClient("127.0.0.1:6553", 10*time.Second)

	// 发送HTTP GET请求到example-service服务的/health端点
	resp, err := client.Get(ctx, "example-service", "/health")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
	} else {
		fmt.Printf("请求成功! 响应: %s\n", string(resp))
	}
}
```

## API参考

### KongDiscoveryClient

服务注册客户端，用于与Kong Discovery API交互。

```go
// 创建客户端
client := sdk.NewClient(registrationURL string)

// 使用默认地址创建客户端 (http://localhost:8081)
client := sdk.NewDefaultClient()

// 注册服务
response, err := client.Register(ctx context.Context, instance *ServiceInstance)

// 注销服务
response, err := client.Deregister(ctx context.Context, serviceName, instanceID string)

// 发送心跳
response, err := client.Heartbeat(ctx context.Context, serviceName, instanceID string, ttl int)

// 启动自动心跳循环
client.StartHeartbeatLoop(ctx context.Context, serviceName, instanceID string, interval time.Duration, ttl int)
```

### DNSDiscovery

DNS服务发现客户端，用于解析服务地址。

```go
// 创建DNS服务发现客户端
discovery := sdk.NewDNSDiscovery(dnsServer string, cacheTTL time.Duration)

// 解析主机地址 (A记录)
host, err := discovery.ResolveHost(ctx context.Context, serviceName string)

// 解析SRV记录
srv, err := discovery.ResolveSRV(ctx context.Context, serviceName string)

// 解析服务（综合A和SRV记录）
address, err := discovery.ResolveService(ctx context.Context, serviceName string)
```

## 最佳实践

1. **实例ID唯一性**: 确保每个服务实例使用唯一的实例ID，可以使用主机名+UUID或时间戳的组合。

2. **心跳间隔**: 心跳间隔建议设置为TTL的1/2或1/3，以确保在网络波动的情况下服务注册不会过期。

3. **优雅退出**: 在应用程序退出前，应该主动注销服务实例，避免服务发现系统中残留过期服务。

4. **错误处理**: 对于服务发现失败的情况，应该有适当的回退策略，如使用缓存的上一次结果或默认值。

5. **缓存TTL**: 根据服务更新频率和稳定性设置合适的缓存TTL，对于稳定的服务可以使用较长的TTL以减少DNS查询次数。

6. **HTTP客户端重试机制**: 在使用基于服务发现的HTTP客户端时，建议实现重试机制，处理服务暂时不可用的情况。

## 故障排除

1. **服务注册失败**:
   - 检查Kong Discovery服务是否正常运行
   - 验证注册URL是否正确
   - 确认提供的服务实例信息是否完整

2. **服务发现失败**:
   - 检查DNS服务器地址是否正确
   - 验证服务是否已成功注册
   - 检查网络连接是否正常

3. **心跳失败**:
   - 检查服务是否已被注册
   - 验证instanceID和serviceName是否匹配
   - 检查网络连接是否稳定

4. **HTTP请求失败**:
   - 检查服务是否已注册且正常运行
   - 验证服务名称是否正确
   - 检查路径是否正确 