package apihandler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
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
	}

	// 测试关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := handler.Shutdown(ctx)
	assert.NoError(t, err)
}
