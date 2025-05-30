package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config SDK客户端配置
type Config struct {
	// 服务发现服务器地址
	ServerAddr string `json:"server_addr"`
	// 服务名称
	ServiceName string `json:"service_name"`
	// 命名空间，默认为"default"
	Namespace string `json:"namespace"`
	// 服务IP地址
	ServiceIP string `json:"service_ip"`
	// 服务端口
	ServicePort int `json:"service_port"`
	// 标签列表
	Tags []string `json:"tags"`
	// 元数据
	Metadata map[string]string `json:"metadata"`
	// 心跳间隔（秒）
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	// 操作超时时间
	Timeout time.Duration `json:"timeout"`
	// 重试次数
	RetryCount int `json:"retry_count"`
	// 是否使用HTTPS
	Secure bool `json:"secure"`
	// API Token（认证使用）
	ApiToken string `json:"api_token"`
}

// Client SDK客户端
type Client struct {
	config       *Config
	httpClient   *http.Client
	serviceID    string
	isRegistered bool
	stopChan     chan struct{}
}

// Response API响应结构
type Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// RegisterResponse 注册响应数据
type RegisterResponse struct {
	ServiceID    string    `json:"service_id"`
	RegisteredAt time.Time `json:"registered_at"`
}

// NewClient 创建SDK客户端
func NewClient(config *Config) (*Client, error) {
	// 验证必填配置
	if config.ServerAddr == "" {
		return nil, fmt.Errorf("服务器地址不能为空")
	}
	if config.ServiceName == "" {
		return nil, fmt.Errorf("服务名称不能为空")
	}
	if config.ServiceIP == "" {
		return nil, fmt.Errorf("服务IP不能为空")
	}
	if config.ServicePort <= 0 {
		return nil, fmt.Errorf("服务端口必须大于0")
	}

	// 设置默认值
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 30 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.Namespace == "" {
		config.Namespace = "default"
	}

	// 创建HTTP客户端
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
		stopChan:   make(chan struct{}),
	}, nil
}

// 构建API地址
func (c *Client) buildURL(path string) string {
	protocol := "http"
	if c.config.Secure {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s%s", protocol, c.config.ServerAddr, path)
}

// 发送HTTP请求
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*Response, error) {
	// 构建URL
	url := c.buildURL(path)

	// 准备请求体
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if c.config.ApiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.ApiToken)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 解析响应
	var apiResp Response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return &apiResp, fmt.Errorf("API请求失败: %s (状态码: %d)", apiResp.Message, resp.StatusCode)
	}

	return &apiResp, nil
}
