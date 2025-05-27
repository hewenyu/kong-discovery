package apihandler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// Handler 定义API处理器接口
type Handler interface {
	// StartManagementAPI 启动管理API服务
	StartManagementAPI() error

	// StartRegistrationAPI 启动服务注册API服务
	StartRegistrationAPI() error

	// Shutdown 优雅关闭API服务
	Shutdown(ctx context.Context) error
}

// EchoHandler 实现Handler接口
type EchoHandler struct {
	managementServer   *echo.Echo
	registrationServer *echo.Echo
	cfg                *config.Config
	logger             config.Logger
}

// NewAPIHandler 创建一个新的API处理器
func NewAPIHandler(cfg *config.Config, logger config.Logger) Handler {
	return &EchoHandler{
		cfg:    cfg,
		logger: logger,
	}
}

// StartManagementAPI 启动管理API服务
func (h *EchoHandler) StartManagementAPI() error {
	h.logger.Info("启动管理API服务",
		zap.String("address", h.cfg.API.Management.ListenAddress),
		zap.Int("port", h.cfg.API.Management.Port))

	// 创建Echo实例
	h.managementServer = echo.New()
	h.managementServer.HideBanner = true

	// 添加中间件
	h.managementServer.Use(middleware.Recover())
	h.managementServer.Use(middleware.Logger())

	// 注册路由
	h.registerManagementRoutes()

	// 启动服务（非阻塞）
	go func() {
		addr := fmt.Sprintf("%s:%d", h.cfg.API.Management.ListenAddress, h.cfg.API.Management.Port)
		if err := h.managementServer.Start(addr); err != nil && err != http.ErrServerClosed {
			h.logger.Error("管理API服务启动失败", zap.Error(err))
		}
	}()

	return nil
}

// StartRegistrationAPI 启动服务注册API服务
func (h *EchoHandler) StartRegistrationAPI() error {
	h.logger.Info("启动服务注册API服务",
		zap.String("address", h.cfg.API.Registration.ListenAddress),
		zap.Int("port", h.cfg.API.Registration.Port))

	// 创建Echo实例
	h.registrationServer = echo.New()
	h.registrationServer.HideBanner = true

	// 添加中间件
	h.registrationServer.Use(middleware.Recover())
	h.registrationServer.Use(middleware.Logger())

	// 注册路由
	h.registerRegistrationRoutes()

	// 启动服务（非阻塞）
	go func() {
		addr := fmt.Sprintf("%s:%d", h.cfg.API.Registration.ListenAddress, h.cfg.API.Registration.Port)
		if err := h.registrationServer.Start(addr); err != nil && err != http.ErrServerClosed {
			h.logger.Error("服务注册API服务启动失败", zap.Error(err))
		}
	}()

	return nil
}

// Shutdown 优雅关闭API服务
func (h *EchoHandler) Shutdown(ctx context.Context) error {
	h.logger.Info("正在关闭API服务...")

	// 关闭管理API服务
	if h.managementServer != nil {
		if err := h.managementServer.Shutdown(ctx); err != nil {
			h.logger.Error("关闭管理API服务出错", zap.Error(err))
			return err
		}
	}

	// 关闭服务注册API服务
	if h.registrationServer != nil {
		if err := h.registrationServer.Shutdown(ctx); err != nil {
			h.logger.Error("关闭服务注册API服务出错", zap.Error(err))
			return err
		}
	}

	return nil
}

// registerManagementRoutes 注册管理API路由
func (h *EchoHandler) registerManagementRoutes() {
	// 健康检查端点
	h.managementServer.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "kong-discovery-management-api",
		})
	})

	// 管理API的其他端点将在后续任务中添加
}

// registerRegistrationRoutes 注册服务注册API路由
func (h *EchoHandler) registerRegistrationRoutes() {
	// 健康检查端点
	h.registrationServer.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "kong-discovery-registration-api",
		})
	})

	// 服务注册API的其他端点将在后续任务中添加
}
