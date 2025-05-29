package handler

import (
	"net/http"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/storage"
	"github.com/labstack/echo/v4"
)

// AdminServiceHandler 管理API服务处理器
type AdminServiceHandler struct {
	storage storage.ServiceStorage
}

// NewAdminServiceHandler 创建管理API服务处理器
func NewAdminServiceHandler(storage storage.ServiceStorage) *AdminServiceHandler {
	return &AdminServiceHandler{
		storage: storage,
	}
}

// ListServices 获取所有服务列表
func (h *AdminServiceHandler) ListServices(c echo.Context) error {
	// 从存储层获取所有服务
	services, err := h.storage.ListServices(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ServiceResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取服务列表失败: " + err.Error(),
		})
	}

	// 返回服务列表
	return c.JSON(http.StatusOK, ServiceResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data: map[string]interface{}{
			"services": services,
		},
	})
}

// GetService 获取服务详情
func (h *AdminServiceHandler) GetService(c echo.Context) error {
	// 获取服务ID
	serviceID := c.Param("serviceId")
	if serviceID == "" {
		return c.JSON(http.StatusBadRequest, ServiceResponse{
			Code:    http.StatusBadRequest,
			Message: "服务ID不能为空",
		})
	}

	// 从存储层获取服务详情
	service, err := h.storage.GetService(c.Request().Context(), serviceID)
	if err != nil {
		// 处理存储层返回的错误
		if se, ok := err.(*storage.StorageError); ok {
			switch se.Code {
			case storage.ErrNotFound:
				return c.JSON(http.StatusNotFound, ServiceResponse{
					Code:    http.StatusNotFound,
					Message: se.Error(),
				})
			default:
				return c.JSON(http.StatusInternalServerError, ServiceResponse{
					Code:    http.StatusInternalServerError,
					Message: "获取服务详情失败: " + se.Error(),
				})
			}
		}

		return c.JSON(http.StatusInternalServerError, ServiceResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取服务详情失败: " + err.Error(),
		})
	}

	// 返回服务详情
	return c.JSON(http.StatusOK, ServiceResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    service,
	})
}

// SystemStatusResponse 系统状态响应
type SystemStatusResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Status      string                 `json:"status"`
		Version     string                 `json:"version"`
		StartTime   time.Time              `json:"start_time"`
		Uptime      string                 `json:"uptime"`
		NumServices int                    `json:"num_services"`
		Resources   map[string]interface{} `json:"resources"`
	} `json:"data"`
}

// GetSystemStatus 获取系统状态
func (h *AdminServiceHandler) GetSystemStatus(c echo.Context) error {
	// 从存储层获取所有服务
	services, err := h.storage.ListServices(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ServiceResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取系统状态失败: " + err.Error(),
		})
	}

	// 创建响应
	response := SystemStatusResponse{
		Code:    http.StatusOK,
		Message: "success",
	}

	// 填充系统状态数据
	response.Data.Status = "running"
	response.Data.Version = "1.0.0"
	response.Data.StartTime = startTime
	response.Data.Uptime = time.Since(startTime).String()
	response.Data.NumServices = len(services)
	response.Data.Resources = getResourceUsage()

	return c.JSON(http.StatusOK, response)
}
