package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	sdk "github.com/hewenyu/kong-discovery/sdk/go"
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
		Timeout:           5 * time.Second,
		RetryCount:        3,
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
	log.Println("服务已启动，按Ctrl+C终止...")
	<-quit

	// 优雅关闭
	log.Println("正在关闭服务...")
	if err := client.Close(ctx); err != nil {
		log.Printf("关闭SDK客户端失败: %v", err)
	}
	log.Println("服务已关闭")
}
