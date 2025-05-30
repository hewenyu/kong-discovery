package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// RegisterRequest 服务注册请求
type RegisterRequest struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	IP        string            `json:"ip"`
	Port      int               `json:"port"`
	Tags      []string          `json:"tags,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	TTL       string            `json:"ttl,omitempty"`
}

// Register 注册服务
func (c *Client) Register(ctx context.Context) error {
	// 判断是否已注册
	if c.isRegistered {
		return fmt.Errorf("服务已注册，服务ID: %s", c.serviceID)
	}

	// 准备请求体
	req := RegisterRequest{
		Name:      c.config.ServiceName,
		Namespace: c.config.Namespace,
		IP:        c.config.ServiceIP,
		Port:      c.config.ServicePort,
		Tags:      c.config.Tags,
		Metadata:  c.config.Metadata,
		TTL:       fmt.Sprintf("%ds", int(c.config.HeartbeatInterval.Seconds())*3), // TTL设置为心跳间隔的3倍
	}

	// 发送注册请求
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/services", req)
	if err != nil {
		return fmt.Errorf("服务注册失败: %w", err)
	}

	// 解析响应
	var registerResp RegisterResponse
	if err := json.Unmarshal(resp.Data, &registerResp); err != nil {
		return fmt.Errorf("解析注册响应失败: %w", err)
	}

	// 保存服务ID
	c.serviceID = registerResp.ServiceID
	c.isRegistered = true

	return nil
}

// Deregister 注销服务
func (c *Client) Deregister(ctx context.Context) error {
	// 判断是否已注册
	if !c.isRegistered {
		return fmt.Errorf("服务尚未注册")
	}

	// 发送注销请求
	_, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/services/%s", c.serviceID), nil)
	if err != nil {
		return fmt.Errorf("服务注销失败: %w", err)
	}

	// 重置状态
	c.isRegistered = false
	c.serviceID = ""

	return nil
}

// GetServiceID 获取服务ID
func (c *Client) GetServiceID() string {
	return c.serviceID
}

// IsRegistered 检查服务是否已注册
func (c *Client) IsRegistered() bool {
	return c.isRegistered
}
