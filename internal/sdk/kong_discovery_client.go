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

// KongDiscoveryClient 是Kong Discovery的SDK客户端
type KongDiscoveryClient struct {
	registrationURL string
	httpClient      *http.Client
}

// ServiceInstance 表示服务实例
type ServiceInstance struct {
	ServiceName string            `json:"service_name"`       // 服务名称
	InstanceID  string            `json:"instance_id"`        // 实例ID
	IPAddress   string            `json:"ip_address"`         // IP地址
	Port        int               `json:"port"`               // 端口
	TTL         int               `json:"ttl"`                // 租约TTL（秒）
	Metadata    map[string]string `json:"metadata,omitempty"` // 可选元数据
}

// DNSRecord 表示DNS记录
type DNSRecord struct {
	Domain string `json:"domain"` // 域名
	Type   string `json:"type"`   // 记录类型 (A, AAAA, CNAME, TXT, SRV等)
	Value  string `json:"value"`  // 记录值
	TTL    int    `json:"ttl"`    // 生存时间（秒）
}

// DNSRecordResponse DNS记录操作响应
type DNSRecordResponse struct {
	Success   bool   `json:"success"`           // 是否成功
	Domain    string `json:"domain"`            // 域名
	Type      string `json:"type"`              // 记录类型
	Message   string `json:"message,omitempty"` // 可选消息
	Timestamp string `json:"timestamp"`         // 时间戳
}

// RegisterResponse 服务注册响应
type RegisterResponse struct {
	Success     bool   `json:"success"`           // 是否成功
	ServiceName string `json:"service_name"`      // 服务名称
	InstanceID  string `json:"instance_id"`       // 实例ID
	Message     string `json:"message,omitempty"` // 可选消息
	Timestamp   string `json:"timestamp"`         // 时间戳
}

// DeregisterResponse 服务注销响应
type DeregisterResponse struct {
	Success     bool   `json:"success"`           // 是否成功
	ServiceName string `json:"service_name"`      // 服务名称
	InstanceID  string `json:"instance_id"`       // 实例ID
	Message     string `json:"message,omitempty"` // 可选消息
	Timestamp   string `json:"timestamp"`         // 时间戳
}

// HeartbeatResponse 服务心跳响应
type HeartbeatResponse struct {
	Success     bool   `json:"success"`           // 是否成功
	ServiceName string `json:"service_name"`      // 服务名称
	InstanceID  string `json:"instance_id"`       // 实例ID
	Message     string `json:"message,omitempty"` // 可选消息
	Timestamp   string `json:"timestamp"`         // 时间戳
}

// HeartbeatRequest 服务心跳请求
type HeartbeatRequest struct {
	TTL int `json:"ttl,omitempty"` // 可选的新TTL值
}

// NewClient 创建Kong Discovery SDK客户端
func NewClient(registrationURL string) *KongDiscoveryClient {
	return &KongDiscoveryClient{
		registrationURL: registrationURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NewDefaultClient 使用默认地址创建客户端
func NewDefaultClient(kongDiscoveryServer string) *KongDiscoveryClient {
	if kongDiscoveryServer == "" {
		kongDiscoveryServer = "http://localhost:8081"
	}
	return NewClient(kongDiscoveryServer)
}

// Register 注册服务实例
func (c *KongDiscoveryClient) Register(ctx context.Context, instance *ServiceInstance) (*RegisterResponse, error) {
	// 验证必要参数
	if instance.ServiceName == "" || instance.InstanceID == "" || instance.IPAddress == "" || instance.Port <= 0 {
		return nil, fmt.Errorf("缺少必要参数：服务名、实例ID、IP地址和端口都是必需的")
	}

	// 设置默认TTL
	if instance.TTL <= 0 {
		instance.TTL = 60 // 默认60秒
	}

	// 序列化请求体
	jsonData, err := json.Marshal(instance)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/services/register", c.registrationURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务注册失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response RegisterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !response.Success {
		return &response, fmt.Errorf("服务注册失败: %s", response.Message)
	}

	return &response, nil
}

// Deregister 注销服务实例
func (c *KongDiscoveryClient) Deregister(ctx context.Context, serviceName, instanceID string) (*DeregisterResponse, error) {
	// 验证必要参数
	if serviceName == "" || instanceID == "" {
		return nil, fmt.Errorf("缺少必要参数：服务名和实例ID都是必需的")
	}

	// 创建请求
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("%s/services/%s/%s", c.registrationURL, serviceName, instanceID),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务注销失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response DeregisterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !response.Success {
		return &response, fmt.Errorf("服务注销失败: %s", response.Message)
	}

	return &response, nil
}

// Heartbeat 发送服务心跳
func (c *KongDiscoveryClient) Heartbeat(ctx context.Context, serviceName, instanceID string, ttl int) (*HeartbeatResponse, error) {
	// 验证必要参数
	if serviceName == "" || instanceID == "" {
		return nil, fmt.Errorf("缺少必要参数：服务名和实例ID都是必需的")
	}

	// 准备请求体
	var reqBody []byte
	var err error
	if ttl > 0 {
		reqBody, err = json.Marshal(HeartbeatRequest{TTL: ttl})
		if err != nil {
			return nil, fmt.Errorf("序列化请求失败: %w", err)
		}
	}

	// 创建请求
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		fmt.Sprintf("%s/services/heartbeat/%s/%s", c.registrationURL, serviceName, instanceID),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	if ttl > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务心跳失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response HeartbeatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !response.Success {
		return &response, fmt.Errorf("服务心跳失败: %s", response.Message)
	}

	return &response, nil
}

// StartHeartbeatLoop 启动心跳循环
func (c *KongDiscoveryClient) StartHeartbeatLoop(ctx context.Context, serviceName, instanceID string, interval time.Duration, ttl int) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 创建新上下文，防止父上下文取消后无法发送最后心跳
				heartbeatCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, err := c.Heartbeat(heartbeatCtx, serviceName, instanceID, ttl)
				cancel()
				if err != nil {
					fmt.Printf("心跳发送失败: %v\n", err)
				}
			}
		}
	}()
}

// CreateDNSRecord 创建DNS记录
func (c *KongDiscoveryClient) CreateDNSRecord(ctx context.Context, record *DNSRecord) (*DNSRecordResponse, error) {
	// 验证必要参数
	if record.Domain == "" || record.Type == "" || record.Value == "" {
		return nil, fmt.Errorf("缺少必要参数：域名、记录类型和记录值都是必需的")
	}

	// 设置默认TTL
	if record.TTL <= 0 {
		record.TTL = 60 // 默认60秒
	}

	// 序列化请求体
	jsonData, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建请求
	// 使用管理API端口（默认8080）
	adminURL := c.registrationURL
	if adminURL == "http://localhost:8081" {
		adminURL = "http://localhost:8080" // 如果使用默认注册API端口，切换到默认管理API端口
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/admin/dns/records", adminURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("创建DNS记录失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response DNSRecordResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !response.Success {
		return &response, fmt.Errorf("创建DNS记录失败: %s", response.Message)
	}

	return &response, nil
}

// DeleteDNSRecord 删除DNS记录
func (c *KongDiscoveryClient) DeleteDNSRecord(ctx context.Context, domain, recordType string) (*DNSRecordResponse, error) {
	// 验证必要参数
	if domain == "" || recordType == "" {
		return nil, fmt.Errorf("缺少必要参数：域名和记录类型都是必需的")
	}

	// 创建请求
	// 使用管理API端口（默认8080）
	adminURL := c.registrationURL
	if adminURL == "http://localhost:8081" {
		adminURL = "http://localhost:8080" // 如果使用默认注册API端口，切换到默认管理API端口
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("%s/admin/dns/records/%s/%s", adminURL, domain, recordType),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("删除DNS记录失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response DNSRecordResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !response.Success {
		return &response, fmt.Errorf("删除DNS记录失败: %s", response.Message)
	}

	return &response, nil
}
