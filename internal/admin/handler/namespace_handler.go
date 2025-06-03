package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hewenyu/kong-discovery/internal/admin/service"
	"github.com/hewenyu/kong-discovery/internal/core/model"
)

// NamespaceHandler 处理命名空间相关的HTTP请求
type NamespaceHandler struct {
	adminService service.AdminService
}

// NewNamespaceHandler 创建一个新的命名空间处理器
func NewNamespaceHandler(adminService service.AdminService) *NamespaceHandler {
	return &NamespaceHandler{
		adminService: adminService,
	}
}

// RegisterRoutes 注册命名空间相关的路由
func (h *NamespaceHandler) RegisterRoutes(e *echo.Echo) {
	api := e.Group("/api/v1")
	api.POST("/namespaces", h.CreateNamespace)
	api.GET("/namespaces", h.ListNamespaces)
	api.GET("/namespaces/:namespace", h.GetNamespaceByName)
	api.DELETE("/namespaces/:namespace", h.DeleteNamespace)
}

// CreateNamespace 创建命名空间
// @Summary 创建命名空间
// @Description 创建一个新的命名空间
// @Tags 命名空间管理
// @Accept json
// @Produce json
// @Param namespace body model.NamespaceCreateRequest true "命名空间创建请求"
// @Success 201 {object} model.ApiResponse
// @Failure 400 {object} model.ApiResponse
// @Failure 500 {object} model.ApiResponse
// @Router /api/v1/namespaces [post]
func (h *NamespaceHandler) CreateNamespace(c echo.Context) error {
	var req model.NamespaceCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ApiResponse{
			Code:    http.StatusBadRequest,
			Message: "请求参数无效",
		})
	}

	// 验证请求参数
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, model.ApiResponse{
			Code:    http.StatusBadRequest,
			Message: "命名空间名称不能为空",
		})
	}

	// 创建命名空间
	namespace := &model.Namespace{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.adminService.CreateNamespace(c.Request().Context(), namespace); err != nil {
		return c.JSON(http.StatusInternalServerError, model.ApiResponse{
			Code:    http.StatusInternalServerError,
			Message: "创建命名空间失败: " + err.Error(),
		})
	}

	// 获取新创建的命名空间完整信息
	createdNamespace, err := h.adminService.GetNamespaceByName(c.Request().Context(), req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.ApiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取命名空间信息失败: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, model.ApiResponse{
		Code:    http.StatusCreated,
		Message: "命名空间创建成功",
		Data:    createdNamespace,
	})
}

// ListNamespaces 获取所有命名空间
// @Summary 获取所有命名空间
// @Description 获取所有命名空间列表
// @Tags 命名空间管理
// @Produce json
// @Success 200 {object} model.ApiResponse{data=model.NamespaceListResponse}
// @Failure 500 {object} model.ApiResponse
// @Router /api/v1/namespaces [get]
func (h *NamespaceHandler) ListNamespaces(c echo.Context) error {
	namespaces, err := h.adminService.ListNamespaces(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.ApiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取命名空间列表失败: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, model.ApiResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data: model.NamespaceListResponse{
			Namespaces: namespaces,
		},
	})
}

// GetNamespaceByName 根据名称获取命名空间
// @Summary 根据名称获取命名空间
// @Description 根据名称获取命名空间详情
// @Tags 命名空间管理
// @Produce json
// @Param namespace path string true "命名空间名称"
// @Success 200 {object} model.ApiResponse{data=model.Namespace}
// @Failure 404 {object} model.ApiResponse
// @Failure 500 {object} model.ApiResponse
// @Router /api/v1/namespaces/{namespace} [get]
func (h *NamespaceHandler) GetNamespaceByName(c echo.Context) error {
	name := c.Param("namespace")
	if name == "" {
		return c.JSON(http.StatusBadRequest, model.ApiResponse{
			Code:    http.StatusBadRequest,
			Message: "命名空间名称不能为空",
		})
	}

	namespace, err := h.adminService.GetNamespaceByName(c.Request().Context(), name)
	if err != nil {
		// 判断是否是命名空间不存在的错误
		if err.Error() == "命名空间不存在: "+name {
			return c.JSON(http.StatusNotFound, model.ApiResponse{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, model.ApiResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取命名空间失败: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, model.ApiResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    namespace,
	})
}

// DeleteNamespace 删除命名空间
// @Summary 删除命名空间
// @Description 删除指定的命名空间
// @Tags 命名空间管理
// @Produce json
// @Param namespace path string true "命名空间名称"
// @Success 200 {object} model.ApiResponse
// @Failure 400 {object} model.ApiResponse
// @Failure 404 {object} model.ApiResponse
// @Failure 500 {object} model.ApiResponse
// @Router /api/v1/namespaces/{namespace} [delete]
func (h *NamespaceHandler) DeleteNamespace(c echo.Context) error {
	name := c.Param("namespace")
	if name == "" {
		return c.JSON(http.StatusBadRequest, model.ApiResponse{
			Code:    http.StatusBadRequest,
			Message: "命名空间名称不能为空",
		})
	}

	err := h.adminService.DeleteNamespace(c.Request().Context(), name)
	if err != nil {
		// 判断错误类型
		if err.Error() == "命名空间不存在: "+name {
			return c.JSON(http.StatusNotFound, model.ApiResponse{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			})
		}
		// 判断是否是因为命名空间有服务导致无法删除
		if err.Error()[:12] == "命名空间包含服务" {
			return c.JSON(http.StatusBadRequest, model.ApiResponse{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, model.ApiResponse{
			Code:    http.StatusInternalServerError,
			Message: "删除命名空间失败: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, model.ApiResponse{
		Code:    http.StatusOK,
		Message: "命名空间删除成功",
	})
}
