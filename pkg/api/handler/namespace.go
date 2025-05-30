package handler

import (
	"net/http"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/storage"
	"github.com/labstack/echo/v4"
)

// NamespaceRequest 命名空间创建请求
type NamespaceRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

// NamespaceResponse 命名空间响应
type NamespaceResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// NamespaceHandler 处理命名空间相关API
type NamespaceHandler struct {
	storage storage.NamespaceStorage
}

// NewNamespaceHandler 创建命名空间处理器
func NewNamespaceHandler(storage storage.NamespaceStorage) *NamespaceHandler {
	return &NamespaceHandler{
		storage: storage,
	}
}

// CreateNamespace 创建命名空间
func (h *NamespaceHandler) CreateNamespace(c echo.Context) error {
	var req NamespaceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NamespaceResponse{
			Code:    http.StatusBadRequest,
			Message: "请求参数无效: " + err.Error(),
		})
	}

	// 参数验证
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NamespaceResponse{
			Code:    http.StatusBadRequest,
			Message: "参数验证失败: " + err.Error(),
		})
	}

	// 创建命名空间实例
	namespace := &storage.Namespace{
		Name:         req.Name,
		Description:  req.Description,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		ServiceCount: 0,
	}

	// 调用存储层创建命名空间
	if err := h.storage.CreateNamespace(c.Request().Context(), namespace); err != nil {
		// 处理存储层返回的错误
		if se, ok := err.(*storage.StorageError); ok {
			switch se.Code {
			case storage.ErrAlreadyExists:
				return c.JSON(http.StatusConflict, NamespaceResponse{
					Code:    http.StatusConflict,
					Message: se.Error(),
				})
			case storage.ErrInvalidArgument:
				return c.JSON(http.StatusBadRequest, NamespaceResponse{
					Code:    http.StatusBadRequest,
					Message: se.Error(),
				})
			default:
				return c.JSON(http.StatusInternalServerError, NamespaceResponse{
					Code:    http.StatusInternalServerError,
					Message: "命名空间创建失败: " + se.Error(),
				})
			}
		}

		return c.JSON(http.StatusInternalServerError, NamespaceResponse{
			Code:    http.StatusInternalServerError,
			Message: "命名空间创建失败: " + err.Error(),
		})
	}

	// 返回成功结果
	return c.JSON(http.StatusOK, NamespaceResponse{
		Code:    http.StatusOK,
		Message: "命名空间创建成功",
		Data: map[string]interface{}{
			"name":       namespace.Name,
			"created_at": namespace.CreatedAt,
		},
	})
}

// DeleteNamespace 删除命名空间
func (h *NamespaceHandler) DeleteNamespace(c echo.Context) error {
	// 获取命名空间名称
	namespaceName := c.Param("namespace")
	if namespaceName == "" {
		return c.JSON(http.StatusBadRequest, NamespaceResponse{
			Code:    http.StatusBadRequest,
			Message: "命名空间名称不能为空",
		})
	}

	// 调用存储层删除命名空间
	if err := h.storage.DeleteNamespace(c.Request().Context(), namespaceName); err != nil {
		// 处理存储层返回的错误
		if se, ok := err.(*storage.StorageError); ok {
			switch se.Code {
			case storage.ErrNotFound:
				return c.JSON(http.StatusNotFound, NamespaceResponse{
					Code:    http.StatusNotFound,
					Message: se.Error(),
				})
			case storage.ErrNamespaceNotEmpty:
				return c.JSON(http.StatusConflict, NamespaceResponse{
					Code:    http.StatusConflict,
					Message: se.Error(),
				})
			default:
				return c.JSON(http.StatusInternalServerError, NamespaceResponse{
					Code:    http.StatusInternalServerError,
					Message: "命名空间删除失败: " + se.Error(),
				})
			}
		}

		return c.JSON(http.StatusInternalServerError, NamespaceResponse{
			Code:    http.StatusInternalServerError,
			Message: "命名空间删除失败: " + err.Error(),
		})
	}

	// 返回成功结果
	return c.JSON(http.StatusOK, NamespaceResponse{
		Code:    http.StatusOK,
		Message: "命名空间删除成功",
	})
}

// GetNamespace 获取命名空间详情
func (h *NamespaceHandler) GetNamespace(c echo.Context) error {
	// 获取命名空间名称
	namespaceName := c.Param("namespace")
	if namespaceName == "" {
		return c.JSON(http.StatusBadRequest, NamespaceResponse{
			Code:    http.StatusBadRequest,
			Message: "命名空间名称不能为空",
		})
	}

	// 调用存储层获取命名空间
	namespace, err := h.storage.GetNamespace(c.Request().Context(), namespaceName)
	if err != nil {
		// 处理存储层返回的错误
		if se, ok := err.(*storage.StorageError); ok {
			switch se.Code {
			case storage.ErrNotFound:
				return c.JSON(http.StatusNotFound, NamespaceResponse{
					Code:    http.StatusNotFound,
					Message: se.Error(),
				})
			default:
				return c.JSON(http.StatusInternalServerError, NamespaceResponse{
					Code:    http.StatusInternalServerError,
					Message: "获取命名空间失败: " + se.Error(),
				})
			}
		}

		return c.JSON(http.StatusInternalServerError, NamespaceResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取命名空间失败: " + err.Error(),
		})
	}

	// 返回成功结果
	return c.JSON(http.StatusOK, NamespaceResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    namespace,
	})
}

// ListNamespaces 获取命名空间列表
func (h *NamespaceHandler) ListNamespaces(c echo.Context) error {
	// 调用存储层获取命名空间列表
	namespaces, err := h.storage.ListNamespaces(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NamespaceResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取命名空间列表失败: " + err.Error(),
		})
	}

	// 返回成功结果
	return c.JSON(http.StatusOK, NamespaceResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data: map[string]interface{}{
			"namespaces": namespaces,
		},
	})
}
