# Kong DNS Discovery Go SDK

本SDK提供了Kong DNS Discovery服务的Go语言客户端，支持服务注册、注销、心跳维持等功能。

## 安装

```bash
go get github.com/hewenyu/kong-discovery/sdk/go
```

## 基本用法

### 创建客户端

```go
import (
    "github.com/hewenyu/kong-discovery/sdk/go"
    "time"
)

// 配置SDK客户端
config := &sdk.Config{
    ServerAddr:        "localhost:8080",        // 服务发现服务器地址
    ServiceName:       "my-service",            // 服务名称
    ServiceIP:         "192.168.1.100",         // 服务IP地址
    ServicePort:       8080,                    // 服务端口
    Tags:              []string{"api", "v1"},   // 标签列表（可选）
    Metadata:          map[string]string{       // 元数据（可选）
        "version": "1.0.0",
    },
    HeartbeatInterval: 30 * time.Second,        // 心跳间隔（可选，默认30秒）
    Timeout:           5 * time.Second,         // 操作超时时间（可选，默认5秒）
    RetryCount:        3,                       // 重试次数（可选，默认3次）
    Secure:            false,                   // 是否使用HTTPS（可选，默认false）
    ApiToken:          "",                      // API Token（可选，用于认证）
}

// 创建SDK客户端
client, err := sdk.NewClient(config)
if err != nil {
    // 处理错误
}
```

### 注册服务

```go
ctx := context.Background()
if err := client.Register(ctx); err != nil {
    // 处理错误
}
fmt.Printf("服务注册成功，服务ID: %s\n", client.GetServiceID())
```

### 自动维持心跳

```go
// 启动心跳任务
client.StartHeartbeat()

// 停止心跳任务
client.StopHeartbeat()
```

### 手动发送心跳

```go
ctx := context.Background()
if err := client.SendHeartbeat(ctx); err != nil {
    // 处理错误
}
```

### 注销服务

```go
ctx := context.Background()
if err := client.Deregister(ctx); err != nil {
    // 处理错误
}
```

### 关闭客户端

```go
// Close会停止心跳并注销服务
ctx := context.Background()
if err := client.Close(ctx); err != nil {
    // 处理错误
}
```

## 完整示例

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/hewenyu/kong-discovery/sdk/go"
)

func main() {
    // 配置SDK客户端
    config := &sdk.Config{
        ServerAddr:        "localhost:8080",
        ServiceName:       "example-service",
        ServiceIP:         "127.0.0.1",
        ServicePort:       8000,
        Tags:              []string{"example", "sdk"},
        Metadata:          map[string]string{"version": "1.0.0"},
        HeartbeatInterval: 30 * time.Second,
    }

    // 创建SDK客户端
    client, err := sdk.NewClient(config)
    if err != nil {
        log.Fatalf("创建SDK客户端失败: %v", err)
    }

    // 注册服务
    ctx := context.Background()
    if err := client.Register(ctx); err != nil {
        log.Fatalf("服务注册失败: %v", err)
    }
    log.Printf("服务注册成功，服务ID: %s", client.GetServiceID())

    // 启动心跳
    client.StartHeartbeat()
    log.Printf("心跳任务已启动，间隔: %s", config.HeartbeatInterval)

    // 等待终止信号
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    // 优雅关闭
    log.Println("正在关闭服务...")
    if err := client.Close(ctx); err != nil {
        log.Printf("关闭SDK客户端失败: %v", err)
    }
    log.Println("服务已关闭")
}
```

## 注意事项

1. 服务注册后，SDK会自动维护心跳，无需手动调用。
2. 使用完毕后，请务必调用`Close()`方法，以确保正确注销服务和停止心跳。
3. 默认心跳间隔为30秒，建议根据服务发现系统的配置适当调整。
4. SDK内部已实现错误重试机制，无需额外处理。 