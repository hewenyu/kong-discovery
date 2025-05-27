package apihandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
	"github.com/hewenyu/kong-discovery/internal/etcdclient"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 创建一个测试用的配置，使用环境变量中的etcd地址
func createTestConfig(t *testing.T) *config.Config {
	t.Helper()

	// 从环境变量中获取etcd地址
	etcdEndpoints := os.Getenv("KONG_DISCOVERY_ETCD_ENDPOINTS")
	require.NotEmpty(t, etcdEndpoints, "环境变量KONG_DISCOVERY_ETCD_ENDPOINTS必须设置")

	// 创建配置
	cfg := &config.Config{}
	cfg.Etcd.Endpoints = []string{etcdEndpoints}
	cfg.Etcd.Username = "" // 如果需要认证，设置相应的值
	cfg.Etcd.Password = "" // 如果需要认证，设置相应的值
	cfg.API.Management.ListenAddress = "localhost"
	cfg.API.Management.Port = 8080
	cfg.API.Registration.ListenAddress = "localhost"
	cfg.API.Registration.Port = 8081

	return cfg
}

// 创建测试用的日志记录器
func createTestLogger(t *testing.T) config.Logger {
	t.Helper()

	logger, err := config.NewLogger(true)
	require.NoError(t, err, "创建测试日志记录器失败")

	return logger
}

// 创建并连接真实的etcd客户端
func createEtcdClient(t *testing.T) etcdclient.Client {
	t.Helper()

	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	client := etcdclient.NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd失败")

	// 确保连接正常
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = client.Ping(ctx)
	require.NoError(t, err, "Ping etcd失败")

	return client
}

// 清理测试数据
func cleanupTestData(t *testing.T, client etcdclient.Client, serviceName, instanceID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 删除测试服务实例
	_ = client.DeregisterService(ctx, serviceName, instanceID)
}

func TestManagementHealthCheck(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建Echo实例和请求
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建handler并注册健康检查路由
	handler := &EchoHandler{
		managementServer: e,
		cfg:              cfg,
		logger:           logger,
		etcdClient:       client,
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
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建Echo实例和请求
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建handler并注册健康检查路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             logger,
		etcdClient:         client,
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
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建Echo实例
	e := echo.New()

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建测试服务名和实例ID，以便测试后清理
	testServiceName := fmt.Sprintf("test-service-%d", time.Now().UnixNano())
	testInstanceID := "instance-001"

	// 确保测试结束后清理数据
	defer cleanupTestData(t, client, testServiceName, testInstanceID)

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             logger,
		etcdClient:         client,
	}
	handler.registerRegistrationRoutes()

	// 准备请求
	reqBody := fmt.Sprintf(`{
		"service_name": "%s",
		"instance_id": "%s",
		"ip_address": "192.168.1.100",
		"port": 8080,
		"ttl": 60,
		"metadata": {
			"version": "1.0.0",
			"region": "cn-north"
		}
	}`, testServiceName, testInstanceID)
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
	assert.Equal(t, testServiceName, response.ServiceName)
	assert.Equal(t, testInstanceID, response.InstanceID)
	assert.Equal(t, "服务注册成功", response.Message)

	// 验证服务是否被保存
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	instances, err := client.GetServiceInstances(ctx, testServiceName)
	require.NoError(t, err)
	assert.Len(t, instances, 1)
	assert.Equal(t, testInstanceID, instances[0].InstanceID)
	assert.Equal(t, "192.168.1.100", instances[0].IPAddress)
	assert.Equal(t, 8080, instances[0].Port)
	assert.Equal(t, 60, instances[0].TTL)
	assert.Equal(t, "1.0.0", instances[0].Metadata["version"])
}

func TestServiceRegistration_BadRequest(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建Echo实例
	e := echo.New()

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             logger,
		etcdClient:         client,
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
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建Echo实例
	e := echo.New()

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建测试服务名和实例ID
	testServiceName := fmt.Sprintf("test-service-%d", time.Now().UnixNano())
	testInstanceID := "instance-001"

	// 先注册一个服务实例
	ctx := context.Background()
	testInstance := &etcdclient.ServiceInstance{
		ServiceName: testServiceName,
		InstanceID:  testInstanceID,
		IPAddress:   "192.168.1.100",
		Port:        8080,
		TTL:         60,
	}
	err := client.RegisterService(ctx, testInstance)
	require.NoError(t, err)

	// 验证服务已注册
	instances, err := client.GetServiceInstances(ctx, testServiceName)
	require.NoError(t, err)
	require.Len(t, instances, 1)

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             logger,
		etcdClient:         client,
	}
	handler.registerRegistrationRoutes()

	// 准备注销请求
	req := httptest.NewRequest(http.MethodDelete, "/services/"+testServiceName+"/"+testInstanceID, nil)
	rec := httptest.NewRecorder()

	// 执行请求
	e.ServeHTTP(rec, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, rec.Code)

	var response ServiceDeregistrationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, testServiceName, response.ServiceName)
	assert.Equal(t, testInstanceID, response.InstanceID)
	assert.Equal(t, "服务注销成功", response.Message)

	// 验证服务已被注销
	instances, err = client.GetServiceInstances(ctx, testServiceName)
	require.NoError(t, err)
	assert.Len(t, instances, 0)
}

func TestServiceDeregistration_NotFound(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建Echo实例
	e := echo.New()

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             logger,
		etcdClient:         client,
	}
	handler.registerRegistrationRoutes()

	// 准备注销请求 - 尝试注销不存在的服务
	nonExistentService := fmt.Sprintf("non-existent-service-%d", time.Now().UnixNano())
	req := httptest.NewRequest(http.MethodDelete, "/services/"+nonExistentService+"/instance-001", nil)
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
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建handler
	managementServer := echo.New()
	registrationServer := echo.New()
	handler := &EchoHandler{
		managementServer:   managementServer,
		registrationServer: registrationServer,
		cfg:                cfg,
		logger:             logger,
		etcdClient:         nil, // 这里不需要实际的etcd客户端
	}

	// 测试关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := handler.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestServiceHeartbeat(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建Echo实例
	e := echo.New()

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建测试服务名和实例ID
	testServiceName := fmt.Sprintf("test-service-%d", time.Now().UnixNano())
	testInstanceID := "instance-001"

	// 确保测试结束后清理数据
	defer cleanupTestData(t, client, testServiceName, testInstanceID)

	// 先注册一个服务实例
	ctx := context.Background()
	testInstance := &etcdclient.ServiceInstance{
		ServiceName: testServiceName,
		InstanceID:  testInstanceID,
		IPAddress:   "192.168.1.100",
		Port:        8080,
		TTL:         60,
	}
	err := client.RegisterService(ctx, testInstance)
	require.NoError(t, err)

	// 验证服务已注册
	instances, err := client.GetServiceInstances(ctx, testServiceName)
	require.NoError(t, err)
	require.Len(t, instances, 1)
	originalTTL := instances[0].TTL

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             logger,
		etcdClient:         client,
	}
	handler.registerRegistrationRoutes()

	// 准备心跳请求 - 更新TTL
	reqBody := `{"ttl": 120}`
	req := httptest.NewRequest(http.MethodPut, "/services/heartbeat/"+testServiceName+"/"+testInstanceID, strings.NewReader(reqBody))
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
	assert.Equal(t, testServiceName, response.ServiceName)
	assert.Equal(t, testInstanceID, response.InstanceID)
	assert.Equal(t, "服务租约刷新成功", response.Message)

	// 验证TTL已更新
	instances, err = client.GetServiceInstances(ctx, testServiceName)
	require.NoError(t, err)
	require.Len(t, instances, 1)
	assert.Equal(t, 120, instances[0].TTL)
	assert.NotEqual(t, originalTTL, instances[0].TTL)
}

func TestServiceHeartbeat_NotFound(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建Echo实例
	e := echo.New()

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建handler并注册路由
	handler := &EchoHandler{
		registrationServer: e,
		cfg:                cfg,
		logger:             logger,
		etcdClient:         client,
	}
	handler.registerRegistrationRoutes()

	// 创建一个随机不存在的服务名
	nonExistentService := fmt.Sprintf("non-existent-service-%d", time.Now().UnixNano())

	// 准备心跳请求 - 对不存在的服务发送心跳
	req := httptest.NewRequest(http.MethodPut, "/services/heartbeat/"+nonExistentService+"/instance-001", nil)
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
