package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

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

// 主函数
func main() {
	// 创建上下文，用于管理整个程序的生命周期
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号，用于优雅退出
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("接收到退出信号，正在优雅退出...")
		cancel()
	}()

	// 创建基于服务发现的HTTP客户端
	client := NewServiceDiscoveryHTTPClient("127.0.0.1:6553", 10*time.Second)

	// 持续尝试发送请求，直到收到退出信号
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <-ctx.Done():
			fmt.Println("程序退出")
			return
		case <-ticker.C:
			count++
			fmt.Printf("\n尝试 #%d:\n", count)

			// 发送HTTP GET请求到example-service服务的/health端点
			resp, err := client.Get(ctx, "example-service", "/health")
			if err != nil {
				fmt.Printf("请求失败: %v\n", err)
			} else {
				fmt.Printf("请求成功! 响应: %s\n", string(resp))
			}
		}
	}
}
