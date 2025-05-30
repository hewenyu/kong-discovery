package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hewenyu/kong-discovery/pkg/storage"
	"github.com/labstack/echo/v4"
)

// ServiceRequest 服务注册请求
type ServiceRequest struct {
	Name      string            `json:"name" validate:"required"`
	Namespace string            `json:"namespace"`
	IP        string            `json:"ip" validate:"required,ip"`
	Port      int               `json:"port" validate:"required,min=1,max=65535"`
	Tags      []string          `json:"tags"`
	Metadata  map[string]string `json:"metadata"`
	TTL       string            `json:"ttl"`
}

// ServiceResponse 服务注册响应
type ServiceResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ServiceHandler 处理服务相关API
type ServiceHandler struct {
	storage storage.ServiceStorage
}

// NewServiceHandler 创建服务处理器
func NewServiceHandler(storage storage.ServiceStorage) *ServiceHandler {
	return &ServiceHandler{
		storage: storage,
	}
}

// RegisterService 注册服务
func (h *ServiceHandler) RegisterService(c echo.Context) error {
	var req ServiceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ServiceResponse{
			Code:    http.StatusBadRequest,
			Message: "请求参数无效: " + err.Error(),
		})
	}

	// 参数验证
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ServiceResponse{
			Code:    http.StatusBadRequest,
			Message: "参数验证失败: " + err.Error(),
		})
	}

	// 生成服务ID
	serviceID := uuid.New().String()

	// 转换TTL
	ttl := 30 // 默认30秒
	if req.TTL != "" {
		duration, err := time.ParseDuration(req.TTL)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ServiceResponse{
				Code:    http.StatusBadRequest,
				Message: "TTL格式无效: " + err.Error(),
			})
		}
		ttl = int(duration.Seconds())
	}

	// 如果未指定命名空间，使用默认命名空间
	if req.Namespace == "" {
		req.Namespace = "default"
	}

	// 创建服务实例
	service := &storage.Service{
		ID:            serviceID,
		Namespace:     req.Namespace,
		Name:          req.Name,
		IP:            req.IP,
		Port:          req.Port,
		Tags:          req.Tags,
		Metadata:      req.Metadata,
		Health:        "healthy",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		TTL:           ttl,
	}

	// 调用存储层注册服务
	if err := h.storage.RegisterService(c.Request().Context(), service); err != nil {
		// 处理存储层返回的错误
		if se, ok := err.(*storage.StorageError); ok {
			switch se.Code {
			case storage.ErrAlreadyExists:
				return c.JSON(http.StatusConflict, ServiceResponse{
					Code:    http.StatusConflict,
					Message: se.Error(),
				})
			case storage.ErrInvalidArgument:
				return c.JSON(http.StatusBadRequest, ServiceResponse{
					Code:    http.StatusBadRequest,
					Message: se.Error(),
				})
			case storage.ErrNotFound:
				return c.JSON(http.StatusNotFound, ServiceResponse{
					Code:    http.StatusNotFound,
					Message: se.Error(),
				})
			default:
				return c.JSON(http.StatusInternalServerError, ServiceResponse{
					Code:    http.StatusInternalServerError,
					Message: "服务注册失败: " + se.Error(),
				})
			}
		}

		return c.JSON(http.StatusInternalServerError, ServiceResponse{
			Code:    http.StatusInternalServerError,
			Message: "服务注册失败: " + err.Error(),
		})
	}

	// 返回成功结果
	return c.JSON(http.StatusOK, ServiceResponse{
		Code:    http.StatusOK,
		Message: "服务注册成功",
		Data: map[string]interface{}{
			"service_id":    serviceID,
			"namespace":     req.Namespace,
			"registered_at": service.RegisteredAt,
		},
	})
}

// DeregisterService 注销服务
func (h *ServiceHandler) DeregisterService(c echo.Context) error {
	// 获取服务ID
	serviceID := c.Param("serviceId")
	if serviceID == "" {
		return c.JSON(http.StatusBadRequest, ServiceResponse{
			Code:    http.StatusBadRequest,
			Message: "服务ID不能为空",
		})
	}

	// 调用存储层注销服务
	if err := h.storage.DeregisterService(c.Request().Context(), serviceID); err != nil {
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
					Message: "服务注销失败: " + se.Error(),
				})
			}
		}

		return c.JSON(http.StatusInternalServerError, ServiceResponse{
			Code:    http.StatusInternalServerError,
			Message: "服务注销失败: " + err.Error(),
		})
	}

	// 返回成功结果
	return c.JSON(http.StatusOK, ServiceResponse{
		Code:    http.StatusOK,
		Message: "服务注销成功",
	})
}

// UpdateHeartbeat 更新服务心跳
func (h *ServiceHandler) UpdateHeartbeat(c echo.Context) error {
	// 获取服务ID
	serviceID := c.Param("serviceId")
	if serviceID == "" {
		return c.JSON(http.StatusBadRequest, ServiceResponse{
			Code:    http.StatusBadRequest,
			Message: "服务ID不能为空",
		})
	}

	// 调用存储层更新心跳
	if err := h.storage.UpdateServiceHeartbeat(c.Request().Context(), serviceID); err != nil {
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
					Message: "心跳更新失败: " + se.Error(),
				})
			}
		}

		return c.JSON(http.StatusInternalServerError, ServiceResponse{
			Code:    http.StatusInternalServerError,
			Message: "心跳更新失败: " + err.Error(),
		})
	}

	// 返回成功结果
	return c.JSON(http.StatusOK, ServiceResponse{
		Code:    http.StatusOK,
		Message: "心跳更新成功",
		Data: map[string]interface{}{
			"last_heartbeat": time.Now(),
		},
	})
}
