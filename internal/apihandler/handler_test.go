package apihandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
	"github.com/hewenyu/kong-discovery/internal/etcdclient"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// MockLogger 实现config.Logger接口，用于测试
type MockLogger struct{}

func (l *MockLogger) Debug(msg string, fields ...zapcore.Field) {}
func (l *MockLogger) Info(msg string, fields ...zapcore.Field)  {}
func (l *MockLogger) Warn(msg string, fields ...zapcore.Field)  {}
func (l *MockLogger) Error(msg string, fields ...zapcore.Field) {}
func (l *MockLogger) Fatal(msg string, fields ...zapcore.Field) {}

// MockEtcdClient 实现etcdclient.Client接口，用于测试
type MockEtcdClient struct {
	services map[string][]*etcdclient.ServiceInstance
}

func NewMockEtcdClient() *MockEtcdClient {
	return &MockEtcdClient{
		services: make(map[string][]*etcdclient.ServiceInstance),
	}
}

func (m *MockEtcdClient) Connect() error                                      { return nil }
func (m *MockEtcdClient) Close() error                                        { return nil }
func (m *MockEtcdClient) Ping(ctx context.Context) error                      { return nil }
func (m *MockEtcdClient) Get(ctx context.Context, key string) (string, error) { return "", nil }
func (m *MockEtcdClient) GetWithPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	return nil, nil
}
func (m *MockEtcdClient) GetDNSRecord(ctx context.Context, domain string, recordType string) (*etcdclient.DNSRecord, error) {
	return nil, nil
}
func (m *MockEtcdClient) PutDNSRecord(ctx context.Context, domain string, record *etcdclient.DNSRecord) error {
	return nil
}
func (m *MockEtcdClient) GetDNSRecordsForDomain(ctx context.Context, domain string) (map[string]*etcdclient.DNSRecord, error) {
	return nil, nil
}

// RegisterService 模拟服务注册
func (m *MockEtcdClient) RegisterService(ctx context.Context, instance *etcdclient.ServiceInstance) error {
	if m.services == nil {
		m.services = make(map[string][]*etcdclient.ServiceInstance)
	}
	m.services[instance.ServiceName] = append(m.services[instance.ServiceName], instance)
	return nil
}

// DeregisterService 模拟服务注销
func (m *MockEtcdClient) DeregisterService(ctx context.Context, serviceName, instanceID string) error {
	instances, ok := m.services[serviceName]
	if !ok {
		return nil
	}

	newInstances := make([]*etcdclient.ServiceInstance, 0, len(instances))
	for _, inst := range instances {
		if inst.InstanceID != instanceID {
			newInstances = append(newInstances, inst)
		}
	}
	m.services[serviceName] = newInstances
	return nil
}

// GetServiceInstances 模拟获取服务实例
func (m *MockEtcdClient) GetServiceInstances(ctx context.Context, serviceName string) ([]*etcdclient.ServiceInstance, error) {
	instances, ok := m.services[serviceName]
	if !ok {
		return nil, nil
	}
	return instances, nil
}

// ServiceToDNSRecords 实现从服务实例到DNS记录的转换
func (m *MockEtcdClient) ServiceToDNSRecords(ctx context.Context, domain string) (map[string]*etcdclient.DNSRecord, error) {
	// 提取服务名（假设domain格式为service.namespace.svc.cluster.local）
	parts := strings.Split(domain, ".")
	if len(parts) < 1 {
		return nil, nil
	}

	serviceName := parts[0]
	instances, _ := m.GetServiceInstances(ctx, serviceName)
	if len(instances) == 0 {
		return nil, nil
	}

	records := make(map[string]*etcdclient.DNSRecord)
	records["A"] = &etcdclient.DNSRecord{
		Type:  "A",
		Value: instances[0].IPAddress,
		TTL:   60,
	}

	for i, instance := range instances {
		records[fmt.Sprintf("SRV-%d", i)] = &etcdclient.DNSRecord{
			Type:  "SRV",
			Value: fmt.Sprintf("10 10 %d %s.%s", instance.Port, instance.InstanceID, domain),
			TTL:   60,
		}
	}

	return records, nil
}

// RefreshServiceLease 模拟刷新服务租约
func (m *MockEtcdClient) RefreshServiceLease(ctx context.Context, serviceName, instanceID string, ttl int) error {
	instances, ok := m.services[serviceName]
	if !ok {
		return fmt.Errorf("服务不存在: %s", serviceName)
	}

	var found bool
	for i, instance := range instances {
		if instance.InstanceID == instanceID {
			// 如果提供了TTL，则更新TTL
			if ttl > 0 {
				instances[i].TTL = ttl
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("实例不存在: %s/%s", serviceName, instanceID)
	}

	return nil
}

func TestManagementHealthCheck(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{}
	cfg.API.Management.ListenAddress = "localhost"
	cfg.API.Management.Port = 8080

	// 创建Echo实例和请求
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// 创建handler并注册健康检查路由
	handler := &EchoHandler{
		managementServer: e,
		cfg:              cfg,
		logger:           &MockLogger{},
		etcdClient:       NewMockEtcdClient(),
	}
	handler.registerManagementRoutes()

	// 执行请求
	e.ServeHTTP(rec, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	assert.Contains(t, response, "timestamp")
	assert.Equal(t, "kong-discovery-management-api", response["service"])
}

func TestRegistrationHealthCheck(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{}
	cfg.API.Registration.ListenAddress = "localhost"
	cfg.API.Registration.Port = 8081

	// 创建Echo实例和请求
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// 创建handler并注册健康检查路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             &MockLogger{},
		etcdClient:         NewMockEtcdClient(),
	}
	handler.registerRegistrationRoutes()

	// 执行请求
	e.ServeHTTP(rec, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	assert.Contains(t, response, "timestamp")
	assert.Equal(t, "kong-discovery-registration-api", response["service"])
}

func TestServiceRegistration(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{}
	cfg.API.Registration.ListenAddress = "localhost"
	cfg.API.Registration.Port = 8081

	// 创建Echo实例
	e := echo.New()

	// 创建模拟etcd客户端
	mockEtcd := NewMockEtcdClient()

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             &MockLogger{},
		etcdClient:         mockEtcd,
	}
	handler.registerRegistrationRoutes()

	// 准备请求
	reqBody := `{
		"service_name": "test-service",
		"instance_id": "instance-001",
		"ip_address": "192.168.1.100",
		"port": 8080,
		"ttl": 60,
		"metadata": {
			"version": "1.0.0",
			"region": "cn-north"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/services/register", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// 执行请求
	e.ServeHTTP(rec, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, rec.Code)

	var response ServiceRegistrationResponse
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, "test-service", response.ServiceName)
	assert.Equal(t, "instance-001", response.InstanceID)
	assert.Equal(t, "服务注册成功", response.Message)

	// 验证服务是否被保存
	instances, err := mockEtcd.GetServiceInstances(context.Background(), "test-service")
	require.NoError(t, err)
	assert.Len(t, instances, 1)
	assert.Equal(t, "instance-001", instances[0].InstanceID)
	assert.Equal(t, "192.168.1.100", instances[0].IPAddress)
	assert.Equal(t, 8080, instances[0].Port)
	assert.Equal(t, 60, instances[0].TTL)
	assert.Equal(t, "1.0.0", instances[0].Metadata["version"])
}

func TestServiceRegistration_BadRequest(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{}
	cfg.API.Registration.ListenAddress = "localhost"
	cfg.API.Registration.Port = 8081

	// 创建Echo实例
	e := echo.New()

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             &MockLogger{},
		etcdClient:         NewMockEtcdClient(),
	}
	handler.registerRegistrationRoutes()

	// 准备请求 - 缺少必要字段
	reqBody := `{
		"service_name": "test-service",
		"instance_id": "instance-001"
		// 缺少 ip_address 和 port
	}`
	req := httptest.NewRequest(http.MethodPost, "/services/register", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// 执行请求
	e.ServeHTTP(rec, req)

	// 验证响应
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response ServiceRegistrationResponse
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Contains(t, response.Message, "请求格式错误")
}

func TestServiceDeregistration(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{}
	cfg.API.Registration.ListenAddress = "localhost"
	cfg.API.Registration.Port = 8081

	// 创建Echo实例
	e := echo.New()

	// 创建模拟etcd客户端
	mockEtcd := NewMockEtcdClient()

	// 先注册一个服务实例
	ctx := context.Background()
	testInstance := &etcdclient.ServiceInstance{
		ServiceName: "test-service",
		InstanceID:  "instance-001",
		IPAddress:   "192.168.1.100",
		Port:        8080,
		TTL:         60,
	}
	err := mockEtcd.RegisterService(ctx, testInstance)
	require.NoError(t, err)

	// 验证服务已注册
	instances, err := mockEtcd.GetServiceInstances(ctx, "test-service")
	require.NoError(t, err)
	require.Len(t, instances, 1)

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             &MockLogger{},
		etcdClient:         mockEtcd,
	}
	handler.registerRegistrationRoutes()

	// 准备注销请求
	req := httptest.NewRequest(http.MethodDelete, "/services/test-service/instance-001", nil)
	rec := httptest.NewRecorder()

	// 执行请求
	e.ServeHTTP(rec, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, rec.Code)

	var response ServiceDeregistrationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, "test-service", response.ServiceName)
	assert.Equal(t, "instance-001", response.InstanceID)
	assert.Equal(t, "服务注销成功", response.Message)

	// 验证服务已被注销
	instances, err = mockEtcd.GetServiceInstances(ctx, "test-service")
	require.NoError(t, err)
	assert.Len(t, instances, 0)
}

func TestServiceDeregistration_NotFound(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{}
	cfg.API.Registration.ListenAddress = "localhost"
	cfg.API.Registration.Port = 8081

	// 创建Echo实例
	e := echo.New()

	// 创建模拟etcd客户端
	mockEtcd := NewMockEtcdClient()

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             &MockLogger{},
		etcdClient:         mockEtcd,
	}
	handler.registerRegistrationRoutes()

	// 准备注销请求 - 尝试注销不存在的服务
	req := httptest.NewRequest(http.MethodDelete, "/services/non-existent-service/instance-001", nil)
	rec := httptest.NewRecorder()

	// 执行请求
	e.ServeHTTP(rec, req)

	// 验证响应 - 即使服务不存在，也应该返回成功
	// 这是因为幂等性原则，多次删除同一资源应该是安全的
	assert.Equal(t, http.StatusOK, rec.Code)

	var response ServiceDeregistrationResponse
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
}

func TestShutdown(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{}
	cfg.API.Management.ListenAddress = "localhost"
	cfg.API.Management.Port = 8080
	cfg.API.Registration.ListenAddress = "localhost"
	cfg.API.Registration.Port = 8081

	// 创建handler
	managementServer := echo.New()
	registrationServer := echo.New()
	handler := &EchoHandler{
		managementServer:   managementServer,
		registrationServer: registrationServer,
		cfg:                cfg,
		logger:             &MockLogger{},
		etcdClient:         NewMockEtcdClient(),
	}

	// 测试关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := handler.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestServiceHeartbeat(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{}
	cfg.API.Registration.ListenAddress = "localhost"
	cfg.API.Registration.Port = 8081

	// 创建Echo实例
	e := echo.New()

	// 创建模拟etcd客户端
	mockEtcd := NewMockEtcdClient()

	// 先注册一个服务实例
	ctx := context.Background()
	testInstance := &etcdclient.ServiceInstance{
		ServiceName: "test-service",
		InstanceID:  "instance-001",
		IPAddress:   "192.168.1.100",
		Port:        8080,
		TTL:         60,
	}
	err := mockEtcd.RegisterService(ctx, testInstance)
	require.NoError(t, err)

	// 验证服务已注册
	instances, err := mockEtcd.GetServiceInstances(ctx, "test-service")
	require.NoError(t, err)
	require.Len(t, instances, 1)
	originalTTL := instances[0].TTL

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             &MockLogger{},
		etcdClient:         mockEtcd,
	}
	handler.registerRegistrationRoutes()

	// 准备心跳请求 - 更新TTL
	reqBody := `{"ttl": 120}`
	req := httptest.NewRequest(http.MethodPut, "/services/heartbeat/test-service/instance-001", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// 执行请求
	e.ServeHTTP(rec, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, rec.Code)

	var response ServiceHeartbeatResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, "test-service", response.ServiceName)
	assert.Equal(t, "instance-001", response.InstanceID)
	assert.Equal(t, "服务租约刷新成功", response.Message)

	// 验证TTL已更新
	instances, err = mockEtcd.GetServiceInstances(ctx, "test-service")
	require.NoError(t, err)
	require.Len(t, instances, 1)
	assert.Equal(t, 120, instances[0].TTL)
	assert.NotEqual(t, originalTTL, instances[0].TTL)
}

func TestServiceHeartbeat_NotFound(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{}
	cfg.API.Registration.ListenAddress = "localhost"
	cfg.API.Registration.Port = 8081

	// 创建Echo实例
	e := echo.New()

	// 创建模拟etcd客户端
	mockEtcd := NewMockEtcdClient()

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             &MockLogger{},
		etcdClient:         mockEtcd,
	}
	handler.registerRegistrationRoutes()

	// 准备心跳请求 - 对不存在的服务发送心跳
	req := httptest.NewRequest(http.MethodPut, "/services/heartbeat/non-existent/instance-001", nil)
	rec := httptest.NewRecorder()

	// 执行请求
	e.ServeHTTP(rec, req)

	// 验证响应 - 应该返回失败
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response ServiceHeartbeatResponse
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Contains(t, response.Message, "刷新服务租约失败")
}
