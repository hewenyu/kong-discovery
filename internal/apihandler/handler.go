package apihandler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
	"github.com/hewenyu/kong-discovery/internal/etcdclient"
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
	etcdClient         etcdclient.Client
}

// NewAPIHandler 创建一个新的API处理器
func NewAPIHandler(cfg *config.Config, logger config.Logger, etcdClient etcdclient.Client) Handler {
	return &EchoHandler{
		cfg:        cfg,
		logger:     logger,
		etcdClient: etcdClient,
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

	// 添加CORS中间件
	h.managementServer.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

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

	// 添加CORS中间件
	h.registrationServer.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

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

	// 获取服务列表端点
	h.managementServer.GET("/admin/services", h.getAllServicesHandler)

	// 获取服务详情端点
	h.managementServer.GET("/admin/services/:serviceName/:instanceId", h.getServiceDetailHandler)
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

	// 服务注册端点
	h.registrationServer.POST("/services/register", h.registerServiceHandler)

	// 服务注销端点
	h.registrationServer.DELETE("/services/:serviceName/:instanceId", h.deregisterServiceHandler)

	// 服务心跳端点
	h.registrationServer.PUT("/services/heartbeat/:serviceName/:instanceId", h.heartbeatServiceHandler)

	// 服务注册API的其他端点将在后续任务中添加
}

// ServiceRegistrationRequest 定义服务注册请求结构
type ServiceRegistrationRequest struct {
	ServiceName string            `json:"service_name" validate:"required"` // 服务名称
	InstanceID  string            `json:"instance_id" validate:"required"`  // 实例ID
	IPAddress   string            `json:"ip_address" validate:"required"`   // IP地址
	Port        int               `json:"port" validate:"required"`         // 端口
	TTL         int               `json:"ttl" validate:"required"`          // 租约TTL（秒）
	Metadata    map[string]string `json:"metadata,omitempty"`               // 可选元数据
}

// ServiceRegistrationResponse 定义服务注册响应结构
type ServiceRegistrationResponse struct {
	Success     bool   `json:"success"`           // 是否成功
	ServiceName string `json:"service_name"`      // 服务名称
	InstanceID  string `json:"instance_id"`       // 实例ID
	Message     string `json:"message,omitempty"` // 可选消息
	Timestamp   string `json:"timestamp"`         // 时间戳
}

// ServiceDeregistrationResponse 定义服务注销响应结构
type ServiceDeregistrationResponse struct {
	Success     bool   `json:"success"`           // 是否成功
	ServiceName string `json:"service_name"`      // 服务名称
	InstanceID  string `json:"instance_id"`       // 实例ID
	Message     string `json:"message,omitempty"` // 可选消息
	Timestamp   string `json:"timestamp"`         // 时间戳
}

// ServiceHeartbeatRequest 定义服务心跳请求结构
type ServiceHeartbeatRequest struct {
	TTL int `json:"ttl,omitempty"` // 可选的新TTL值
}

// ServiceHeartbeatResponse 定义服务心跳响应结构
type ServiceHeartbeatResponse struct {
	Success     bool   `json:"success"`           // 是否成功
	ServiceName string `json:"service_name"`      // 服务名称
	InstanceID  string `json:"instance_id"`       // 实例ID
	Message     string `json:"message,omitempty"` // 可选消息
	Timestamp   string `json:"timestamp"`         // 时间戳
}

// registerServiceHandler 处理服务注册请求
func (h *EchoHandler) registerServiceHandler(c echo.Context) error {
	// 解析请求
	req := new(ServiceRegistrationRequest)
	if err := c.Bind(req); err != nil {
		h.logger.Error("解析服务注册请求失败", zap.Error(err))
		return c.JSON(http.StatusBadRequest, &ServiceRegistrationResponse{
			Success:   false,
			Message:   "请求格式错误: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// 验证请求
	if req.ServiceName == "" || req.InstanceID == "" || req.IPAddress == "" || req.Port <= 0 {
		h.logger.Warn("服务注册请求参数无效",
			zap.String("service", req.ServiceName),
			zap.String("id", req.InstanceID))
		return c.JSON(http.StatusBadRequest, &ServiceRegistrationResponse{
			Success:   false,
			Message:   "请求参数无效：服务名、实例ID、IP地址和端口都是必需的",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// 设置默认TTL
	if req.TTL <= 0 {
		req.TTL = 60 // 默认60秒
	}

	// 转换为服务实例
	instance := &etcdclient.ServiceInstance{
		ServiceName: req.ServiceName,
		InstanceID:  req.InstanceID,
		IPAddress:   req.IPAddress,
		Port:        req.Port,
		Metadata:    req.Metadata,
		TTL:         req.TTL,
	}

	// 注册服务
	ctx := c.Request().Context()
	err := h.etcdClient.RegisterService(ctx, instance)
	if err != nil {
		h.logger.Error("注册服务实例失败",
			zap.String("service", req.ServiceName),
			zap.String("id", req.InstanceID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &ServiceRegistrationResponse{
			Success:     false,
			ServiceName: req.ServiceName,
			InstanceID:  req.InstanceID,
			Message:     "注册服务失败: " + err.Error(),
			Timestamp:   time.Now().Format(time.RFC3339),
		})
	}

	// 返回成功响应
	h.logger.Info("服务注册成功",
		zap.String("service", req.ServiceName),
		zap.String("id", req.InstanceID))
	return c.JSON(http.StatusOK, &ServiceRegistrationResponse{
		Success:     true,
		ServiceName: req.ServiceName,
		InstanceID:  req.InstanceID,
		Message:     "服务注册成功",
		Timestamp:   time.Now().Format(time.RFC3339),
	})
}

// deregisterServiceHandler 处理服务注销请求
func (h *EchoHandler) deregisterServiceHandler(c echo.Context) error {
	// 从URL参数中获取服务名和实例ID
	serviceName := c.Param("serviceName")
	instanceID := c.Param("instanceId")

	// 验证参数
	if serviceName == "" || instanceID == "" {
		h.logger.Warn("服务注销请求参数无效",
			zap.String("service", serviceName),
			zap.String("id", instanceID))
		return c.JSON(http.StatusBadRequest, &ServiceDeregistrationResponse{
			Success:   false,
			Message:   "请求参数无效：服务名和实例ID都是必需的",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// 从etcd中注销服务
	ctx := c.Request().Context()
	err := h.etcdClient.DeregisterService(ctx, serviceName, instanceID)
	if err != nil {
		h.logger.Error("注销服务实例失败",
			zap.String("service", serviceName),
			zap.String("id", instanceID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &ServiceDeregistrationResponse{
			Success:     false,
			ServiceName: serviceName,
			InstanceID:  instanceID,
			Message:     "注销服务失败: " + err.Error(),
			Timestamp:   time.Now().Format(time.RFC3339),
		})
	}

	// 返回成功响应
	h.logger.Info("服务注销成功",
		zap.String("service", serviceName),
		zap.String("id", instanceID))
	return c.JSON(http.StatusOK, &ServiceDeregistrationResponse{
		Success:     true,
		ServiceName: serviceName,
		InstanceID:  instanceID,
		Message:     "服务注销成功",
		Timestamp:   time.Now().Format(time.RFC3339),
	})
}

// heartbeatServiceHandler 处理服务心跳请求
func (h *EchoHandler) heartbeatServiceHandler(c echo.Context) error {
	// 从URL参数中获取服务名和实例ID
	serviceName := c.Param("serviceName")
	instanceID := c.Param("instanceId")

	// 验证参数
	if serviceName == "" || instanceID == "" {
		h.logger.Warn("服务心跳请求参数无效",
			zap.String("service", serviceName),
			zap.String("id", instanceID))
		return c.JSON(http.StatusBadRequest, &ServiceHeartbeatResponse{
			Success:   false,
			Message:   "请求参数无效：服务名和实例ID都是必需的",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// 解析请求体中的TTL（如果有）
	var req ServiceHeartbeatRequest
	var ttl int
	if err := c.Bind(&req); err == nil && req.TTL > 0 {
		ttl = req.TTL
	}

	// 刷新服务实例的租约
	ctx := c.Request().Context()
	err := h.etcdClient.RefreshServiceLease(ctx, serviceName, instanceID, ttl)
	if err != nil {
		h.logger.Error("刷新服务实例租约失败",
			zap.String("service", serviceName),
			zap.String("id", instanceID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &ServiceHeartbeatResponse{
			Success:     false,
			ServiceName: serviceName,
			InstanceID:  instanceID,
			Message:     "刷新服务租约失败: " + err.Error(),
			Timestamp:   time.Now().Format(time.RFC3339),
		})
	}

	// 返回成功响应
	h.logger.Info("服务心跳成功",
		zap.String("service", serviceName),
		zap.String("id", instanceID))
	return c.JSON(http.StatusOK, &ServiceHeartbeatResponse{
		Success:     true,
		ServiceName: serviceName,
		InstanceID:  instanceID,
		Message:     "服务租约刷新成功",
		Timestamp:   time.Now().Format(time.RFC3339),
	})
}

// ServiceListResponse 定义服务列表响应结构
type ServiceListResponse struct {
	Success   bool     `json:"success"`           // 是否成功
	Services  []string `json:"services"`          // 服务名称列表
	Message   string   `json:"message,omitempty"` // 可选消息
	Count     int      `json:"count"`             // 服务数量
	Timestamp string   `json:"timestamp"`         // 时间戳
}

// ServiceDetailResponse 定义服务详情响应结构
type ServiceDetailResponse struct {
	Success     bool              `json:"success"`            // 是否成功
	ServiceName string            `json:"service_name"`       // 服务名称
	InstanceID  string            `json:"instance_id"`        // 实例ID
	IPAddress   string            `json:"ip_address"`         // IP地址
	Port        int               `json:"port"`               // 端口
	TTL         int               `json:"ttl"`                // TTL（秒）
	Metadata    map[string]string `json:"metadata,omitempty"` // 可选元数据
	Message     string            `json:"message,omitempty"`  // 可选消息
	Timestamp   string            `json:"timestamp"`          // 时间戳
}

// getAllServicesHandler 处理获取所有服务列表的请求
func (h *EchoHandler) getAllServicesHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// 从etcd获取所有服务名称
	serviceNames, err := h.etcdClient.GetAllServiceNames(ctx)
	if err != nil {
		h.logger.Error("获取服务列表失败", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &ServiceListResponse{
			Success:   false,
			Message:   "获取服务列表失败: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// 返回服务列表
	return c.JSON(http.StatusOK, &ServiceListResponse{
		Success:   true,
		Services:  serviceNames,
		Count:     len(serviceNames),
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// getServiceDetailHandler 处理获取服务实例详情的请求
func (h *EchoHandler) getServiceDetailHandler(c echo.Context) error {
	// 获取路径参数
	serviceName := c.Param("serviceName")
	instanceID := c.Param("instanceId")

	// 验证参数
	if serviceName == "" || instanceID == "" {
		h.logger.Warn("服务详情请求参数无效")
		return c.JSON(http.StatusBadRequest, &ServiceDetailResponse{
			Success:   false,
			Message:   "请求参数无效：服务名和实例ID都是必需的",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	ctx := c.Request().Context()

	// 获取服务实例列表
	instances, err := h.etcdClient.GetServiceInstances(ctx, serviceName)
	if err != nil {
		h.logger.Error("获取服务实例列表失败",
			zap.String("service", serviceName),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &ServiceDetailResponse{
			Success:     false,
			ServiceName: serviceName,
			InstanceID:  instanceID,
			Message:     "获取服务实例列表失败: " + err.Error(),
			Timestamp:   time.Now().Format(time.RFC3339),
		})
	}

	// 查找指定的实例
	var targetInstance *etcdclient.ServiceInstance
	for _, instance := range instances {
		if instance.InstanceID == instanceID {
			targetInstance = instance
			break
		}
	}

	// 如果未找到实例
	if targetInstance == nil {
		h.logger.Warn("未找到指定的服务实例",
			zap.String("service", serviceName),
			zap.String("id", instanceID))
		return c.JSON(http.StatusNotFound, &ServiceDetailResponse{
			Success:     false,
			ServiceName: serviceName,
			InstanceID:  instanceID,
			Message:     "未找到指定的服务实例",
			Timestamp:   time.Now().Format(time.RFC3339),
		})
	}

	// 返回实例详情
	return c.JSON(http.StatusOK, &ServiceDetailResponse{
		Success:     true,
		ServiceName: targetInstance.ServiceName,
		InstanceID:  targetInstance.InstanceID,
		IPAddress:   targetInstance.IPAddress,
		Port:        targetInstance.Port,
		TTL:         targetInstance.TTL,
		Metadata:    targetInstance.Metadata,
		Timestamp:   time.Now().Format(time.RFC3339),
	})
}
