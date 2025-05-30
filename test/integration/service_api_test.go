package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hewenyu/kong-discovery/pkg/api/handler"
	"github.com/hewenyu/kong-discovery/pkg/api/router"
	"github.com/hewenyu/kong-discovery/pkg/config"
	"github.com/hewenyu/kong-discovery/pkg/storage"
	"github.com/hewenyu/kong-discovery/pkg/storage/etcd"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
)

// 测试服务器和存储
type TestServer struct {
	Echo             *echo.Echo
	Storage          storage.ServiceStorage
	NamespaceStorage storage.NamespaceStorage
	Server           *httptest.Server
	Services         map[string]*storage.Service // 用于跟踪注册的服务
	EtcdClient       *etcd.Client                // 存储etcd客户端，用于测试后清理
}

// 自定义验证器
type CustomValidator struct {
	validator *validator.Validate
}

// Validate 实现echo.Validator接口
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// 创建测试服务器
func NewTestServer() *TestServer {
	var serviceStorage storage.ServiceStorage
	var namespaceStorage storage.NamespaceStorage
	var etcdClient *etcd.Client

	// 从环境变量读取 etcd 配置
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		etcdEndpoints = "localhost:2379" // 默认连接本地 etcd
	}

	etcdConfig := &config.EtcdConfig{
		Endpoints:   strings.Split(etcdEndpoints, ","),
		DialTimeout: os.Getenv("ETCD_DIAL_TIMEOUT"),
		Username:    os.Getenv("ETCD_USERNAME"),
		Password:    os.Getenv("ETCD_PASSWORD"),
	}

	// 默认超时时间
	if etcdConfig.DialTimeout == "" {
		etcdConfig.DialTimeout = "10s"
	}

	// 创建 etcd 客户端
	client, err := etcd.NewClient(etcdConfig)
	if err != nil {
		// 如果连接 etcd 失败，直接跳过测试
		fmt.Printf("无法连接到 etcd: %v\n", err)
		os.Exit(1)
	}

	etcdClient = client
	// 使用 etcd 存储
	serviceStorage = etcd.NewServiceStorage(client)
	namespaceStorage = etcd.NewNamespaceStorage(client)

	// 创建默认命名空间
	ctx := context.Background()
	defaultNs := &storage.Namespace{
		Name:        "default",
		Description: "默认命名空间",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = namespaceStorage.CreateNamespace(ctx, defaultNs)
	// 忽略已存在的命名空间错误
	if err != nil {
		if se, ok := err.(*storage.StorageError); !ok || se.Code != storage.ErrAlreadyExists {
			fmt.Printf("创建默认命名空间失败: %v\n", err)
		}
	}

	// 创建Echo实例
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// 创建处理器
	serviceHandler := handler.NewServiceHandler(serviceStorage)
	healthHandler := handler.NewHealthHandler(serviceStorage)

	// 注册路由
	router.RegisterRoutes(e, serviceHandler, healthHandler)

	// 创建HTTP测试服务器
	server := httptest.NewServer(e)

	return &TestServer{
		Echo:             e,
		Storage:          serviceStorage,
		NamespaceStorage: namespaceStorage,
		Server:           server,
		Services:         make(map[string]*storage.Service),
		EtcdClient:       etcdClient,
	}
}

// 关闭测试服务器
func (ts *TestServer) Close() {
	ts.Server.Close()
	if ts.EtcdClient != nil {
		ts.EtcdClient.Close()
	}
}

// TestMain 测试主函数
func TestMain(m *testing.M) {
	// 设置测试环境
	os.Setenv("KONG_DISCOVERY_SERVER_REGISTER_PORT", "8081")

	// 检查是否设置了 ETCD_ENDPOINTS
	if os.Getenv("ETCD_ENDPOINTS") == "" {
		fmt.Println("警告: ETCD_ENDPOINTS 环境变量未设置，将使用默认值 localhost:2379")
	}

	// 运行测试
	code := m.Run()

	// 清理测试环境
	os.Unsetenv("KONG_DISCOVERY_SERVER_REGISTER_PORT")

	os.Exit(code)
}

// 测试服务注册API
func TestRegisterService(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	// 测试有效请求
	t.Run("Valid Registration", func(t *testing.T) {
		// 创建上下文
		ctx := context.Background()

		// 创建请求体
		reqBody := map[string]interface{}{
			"name":      "test-service",
			"namespace": "default",
			"ip":        "192.168.1.100",
			"port":      8080,
			"tags":      []string{"test", "api"},
			"metadata":  map[string]string{"version": "1.0.0"},
			"ttl":       "30s",
		}
		jsonData, _ := json.Marshal(reqBody)

		// 发送请求
		resp, err := http.Post(
			ts.Server.URL+"/api/v1/services",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 解析响应
		var result handler.ServiceResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, result.Code)
		assert.Contains(t, result.Message, "服务注册成功")

		// 验证数据
		data := result.Data.(map[string]interface{})
		serviceID := data["service_id"].(string)
		assert.NotEmpty(t, serviceID)

		// 验证服务已注册到存储中
		service, err := ts.Storage.GetService(ctx, serviceID)
		assert.NoError(t, err)
		assert.Equal(t, "test-service", service.Name)
		assert.Equal(t, "192.168.1.100", service.IP)
		assert.Equal(t, 8080, service.Port)
		assert.Equal(t, "default", service.Namespace)

		// 保存服务ID用于后续测试
		ts.Services["test-service"] = service
	})

	// 测试无效请求 - 缺少必填字段
	t.Run("Invalid Registration - Missing Required Fields", func(t *testing.T) {
		// 创建请求体 - 缺少IP字段
		reqBody := map[string]interface{}{
			"name": "invalid-service",
			"port": 8080,
		}
		jsonData, _ := json.Marshal(reqBody)

		// 发送请求
		resp, err := http.Post(
			ts.Server.URL+"/api/v1/services",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// 解析响应
		var result handler.ServiceResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, result.Code)
		assert.Contains(t, result.Message, "参数验证失败")
	})

	// 测试无效请求 - 无效的IP地址
	t.Run("Invalid Registration - Invalid IP", func(t *testing.T) {
		// 创建请求体 - 无效的IP地址
		reqBody := map[string]interface{}{
			"name": "invalid-ip-service",
			"ip":   "999.999.999.999", // 无效IP
			"port": 8080,
		}
		jsonData, _ := json.Marshal(reqBody)

		// 发送请求
		resp, err := http.Post(
			ts.Server.URL+"/api/v1/services",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// 解析响应
		var result handler.ServiceResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, result.Code)
		assert.Contains(t, result.Message, "参数验证失败")
	})

	// 测试无效请求 - 无效的TTL格式
	t.Run("Invalid Registration - Invalid TTL", func(t *testing.T) {
		// 创建请求体 - 无效的TTL格式
		reqBody := map[string]interface{}{
			"name": "invalid-ttl-service",
			"ip":   "192.168.1.100",
			"port": 8080,
			"ttl":  "invalid", // 无效TTL格式
		}
		jsonData, _ := json.Marshal(reqBody)

		// 发送请求
		resp, err := http.Post(
			ts.Server.URL+"/api/v1/services",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// 解析响应
		var result handler.ServiceResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, result.Code)
		assert.Contains(t, result.Message, "TTL格式无效")
	})
}

// 测试服务注销API
func TestDeregisterService(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	// 创建上下文
	ctx := context.Background()

	// 先注册一个服务
	service := &storage.Service{
		ID:            "test-service-id",
		Name:          "test-deregister-service",
		Namespace:     "default",
		IP:            "192.168.1.100",
		Port:          8080,
		Health:        "healthy",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		TTL:           30,
	}
	err := ts.Storage.RegisterService(ctx, service)
	assert.NoError(t, err)
	ts.Services["test-deregister-service"] = service

	// 测试有效的注销请求
	t.Run("Valid Deregistration", func(t *testing.T) {
		// 创建DELETE请求
		req, err := http.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("%s/api/v1/services/%s", ts.Server.URL, service.ID),
			nil,
		)
		assert.NoError(t, err)

		// 发送请求
		client := http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 解析响应
		var result handler.ServiceResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, result.Code)
		assert.Contains(t, result.Message, "服务注销成功")

		// 验证服务已从存储中删除
		_, err = ts.Storage.GetService(ctx, service.ID)
		assert.Error(t, err) // 应该返回错误，因为服务已被删除
	})

	// 测试注销不存在的服务
	t.Run("Deregister Non-existent Service", func(t *testing.T) {
		// 创建DELETE请求
		req, err := http.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("%s/api/v1/services/non-existent-id", ts.Server.URL),
			nil,
		)
		assert.NoError(t, err)

		// 发送请求
		client := http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// 解析响应
		var result handler.ServiceResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, result.Code)
	})
}

// 测试心跳更新API
func TestHeartbeatUpdate(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	// 创建上下文
	ctx := context.Background()

	// 先注册一个服务
	service := &storage.Service{
		ID:            "test-heartbeat-id",
		Name:          "test-heartbeat-service",
		Namespace:     "default",
		IP:            "192.168.1.100",
		Port:          8080,
		Health:        "healthy",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now().Add(-10 * time.Second), // 设置上次心跳为10秒前
		TTL:           30,
	}
	err := ts.Storage.RegisterService(ctx, service)
	assert.NoError(t, err)
	ts.Services["test-heartbeat-service"] = service

	// 测试有效的心跳更新
	t.Run("Valid Heartbeat Update", func(t *testing.T) {
		// 记录原始心跳时间
		originalHeartbeat := service.LastHeartbeat

		// 创建PUT请求
		req, err := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("%s/api/v1/services/%s/heartbeat", ts.Server.URL, service.ID),
			nil,
		)
		assert.NoError(t, err)

		// 发送请求
		client := http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 解析响应
		var result handler.ServiceResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, result.Code)
		assert.Contains(t, result.Message, "心跳更新成功")

		// 验证心跳时间已更新
		updatedService, err := ts.Storage.GetService(ctx, service.ID)
		assert.NoError(t, err)
		assert.True(t, updatedService.LastHeartbeat.After(originalHeartbeat))
	})

	// 测试更新不存在服务的心跳
	t.Run("Heartbeat Update Non-existent Service", func(t *testing.T) {
		// 创建PUT请求
		req, err := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("%s/api/v1/services/non-existent-id/heartbeat", ts.Server.URL),
			nil,
		)
		assert.NoError(t, err)

		// 发送请求
		client := http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// 解析响应
		var result handler.ServiceResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, result.Code)
	})
}

// 测试健康检查API
func TestHealthCheck(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	// 使用带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 检查默认命名空间是否存在
	_, err := ts.NamespaceStorage.GetNamespace(ctx, "default")
	assert.NoError(t, err, "默认命名空间应该已创建")

	t.Run("Health Check", func(t *testing.T) {
		// 发送请求
		resp, err := http.Get(ts.Server.URL + "/api/v1/health")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 解析响应
		var result handler.HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, "healthy", result.Status)
		assert.NotNil(t, result.Details)
	})
}
