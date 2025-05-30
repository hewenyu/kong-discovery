package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hewenyu/kong-discovery/internal/admin/service"
	"github.com/hewenyu/kong-discovery/internal/core/model"
)

// ServiceHandler 处理服务管理相关的HTTP请求
type ServiceHandler struct {
	service service.AdminService
}

// NewServiceHandler 创建一个新的服务管理处理器
func NewServiceHandler(service service.AdminService) *ServiceHandler {
	return &ServiceHandler{
		service: service,
	}
}

// RegisterRoutes 注册API路由
func (h *ServiceHandler) RegisterRoutes(e *echo.Echo) {
	api := e.Group("/api/v1")

	// 查询服务列表
	api.GET("/services", h.listServices)

	// 查询服务详情
	api.GET("/services/:serviceId", h.getServiceByID)
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

// listServices 处理查询服务列表请求
func (h *ServiceHandler) listServices(c echo.Context) error {
	// 获取查询参数
	namespace := c.QueryParam("namespace")

	// 调用服务层查询服务列表
	services, err := h.service.ListServices(c.Request().Context(), namespace)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(http.StatusInternalServerError, "查询服务列表失败: "+err.Error()))
	}

	// 构造响应数据
	data := map[string]interface{}{
		"services": services,
	}

	// 返回成功响应
	return c.JSON(http.StatusOK, successResponse(http.StatusOK, "查询成功", data))
}

// getServiceByID 处理查询服务详情请求
func (h *ServiceHandler) getServiceByID(c echo.Context) error {
	// 获取服务ID
	serviceID := c.Param("serviceId")
	if serviceID == "" {
		return c.JSON(http.StatusBadRequest, errorResponse(http.StatusBadRequest, "服务ID不能为空"))
	}

	// 调用服务层查询服务详情
	service, err := h.service.GetServiceByID(c.Request().Context(), serviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(http.StatusInternalServerError, "查询服务详情失败: "+err.Error()))
	}

	// 返回成功响应
	return c.JSON(http.StatusOK, successResponse(http.StatusOK, "查询成功", service))
}
