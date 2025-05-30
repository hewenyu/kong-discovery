package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hewenyu/kong-discovery/internal/core/model"
	"github.com/hewenyu/kong-discovery/internal/registration/service"
)

// RegistrationHandler 处理服务注册相关的HTTP请求
type RegistrationHandler struct {
	service service.RegistrationService
}

// NewRegistrationHandler 创建一个新的服务注册处理器
func NewRegistrationHandler(service service.RegistrationService) *RegistrationHandler {
	return &RegistrationHandler{
		service: service,
	}
}

// RegisterRoutes 注册API路由
func (h *RegistrationHandler) RegisterRoutes(e *echo.Echo) {
	api := e.Group("/api/v1")

	// 服务注册
	api.POST("/services", h.registerService)

	// 服务注销
	api.DELETE("/services/:serviceId", h.deregisterService)

	// 服务心跳
	api.PUT("/services/:serviceId/heartbeat", h.updateHeartbeat)
}

// 返回成功响应
func successResponse(code int, message string, data interface{}) *model.ApiResponse {
	return &model.ApiResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// 返回错误响应
func errorResponse(code int, message string) *model.ApiResponse {
	return &model.ApiResponse{
		Code:    code,
		Message: message,
	}
}

// registerService 处理服务注册请求
func (h *RegistrationHandler) registerService(c echo.Context) error {
	// 解析请求参数
	req := new(model.ServiceRegistrationRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(http.StatusBadRequest, "无效的请求参数: "+err.Error()))
	}

	// 校验必填字段
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, errorResponse(http.StatusBadRequest, "服务名称不能为空"))
	}
	if req.IP == "" {
		return c.JSON(http.StatusBadRequest, errorResponse(http.StatusBadRequest, "服务IP不能为空"))
	}
	if req.Port <= 0 || req.Port > 65535 {
		return c.JSON(http.StatusBadRequest, errorResponse(http.StatusBadRequest, "无效的服务端口"))
	}

	// 调用服务层注册服务
	resp, err := h.service.RegisterService(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(http.StatusInternalServerError, "注册服务失败: "+err.Error()))
	}

	// 返回成功响应
	return c.JSON(http.StatusOK, successResponse(http.StatusOK, "服务注册成功", resp))
}

// deregisterService 处理服务注销请求
func (h *RegistrationHandler) deregisterService(c echo.Context) error {
	// 获取服务ID
	serviceID := c.Param("serviceId")
	if serviceID == "" {
		return c.JSON(http.StatusBadRequest, errorResponse(http.StatusBadRequest, "服务ID不能为空"))
	}

	// 调用服务层注销服务
	if err := h.service.DeregisterService(c.Request().Context(), serviceID); err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(http.StatusInternalServerError, "注销服务失败: "+err.Error()))
	}

	// 返回成功响应
	return c.JSON(http.StatusOK, successResponse(http.StatusOK, "服务注销成功", nil))
}

// updateHeartbeat 处理服务心跳请求
func (h *RegistrationHandler) updateHeartbeat(c echo.Context) error {
	// 获取服务ID
	serviceID := c.Param("serviceId")
	if serviceID == "" {
		return c.JSON(http.StatusBadRequest, errorResponse(http.StatusBadRequest, "服务ID不能为空"))
	}

	// 调用服务层更新心跳
	resp, err := h.service.UpdateHeartbeat(c.Request().Context(), serviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(http.StatusInternalServerError, "更新心跳失败: "+err.Error()))
	}

	// 返回成功响应
	return c.JSON(http.StatusOK, successResponse(http.StatusOK, "心跳更新成功", resp))
}

// StartCleanupTask 启动定期清理任务
func (h *RegistrationHandler) StartCleanupTask(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			count, err := h.service.CleanupStaleServices(context.Background())
			if err != nil {
				log.Printf("清理过期服务失败: %v", err)
			} else if count > 0 {
				log.Printf("清理了 %d 个过期服务", count)
			}
		}
	}()
}
