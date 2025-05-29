package sdk

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// SendHeartbeat 发送心跳
func (c *Client) SendHeartbeat(ctx context.Context) error {
	// 判断是否已注册
	if !c.isRegistered {
		return fmt.Errorf("服务尚未注册")
	}

	// 发送心跳请求
	_, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1/services/%s/heartbeat", c.serviceID), nil)
	if err != nil {
		return fmt.Errorf("发送心跳失败: %w", err)
	}

	return nil
}

// StartHeartbeat 开始心跳任务
func (c *Client) StartHeartbeat() {
	// 停止已有心跳任务
	c.StopHeartbeat()

	// 创建新的停止通道
	c.stopChan = make(chan struct{})

	// 启动心跳协程
	go func() {
		ticker := time.NewTicker(c.config.HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 创建超时上下文
				ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)

				// 发送心跳
				if err := c.SendHeartbeat(ctx); err != nil {
					log.Printf("心跳发送失败: %v, 将在下一个周期重试", err)
				}

				cancel() // 取消上下文
			case <-c.stopChan:
				return
			}
		}
	}()
}

// StopHeartbeat 停止心跳任务
func (c *Client) StopHeartbeat() {
	if c.stopChan != nil {
		close(c.stopChan)
	}
}

// Close 关闭客户端
func (c *Client) Close(ctx context.Context) error {
	// 停止心跳任务
	c.StopHeartbeat()

	// 如果已注册，注销服务
	if c.isRegistered {
		if err := c.Deregister(ctx); err != nil {
			return fmt.Errorf("注销服务失败: %w", err)
		}
	}

	return nil
}
