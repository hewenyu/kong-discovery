package namespace

import (
	"context"

	"github.com/hewenyu/kong-discovery/internal/core/model"
)

// NamespaceStore 定义命名空间存储接口
type NamespaceStore interface {
	// CreateNamespace 创建命名空间
	CreateNamespace(ctx context.Context, namespace *model.Namespace) error

	// GetNamespaceByName 根据名称获取命名空间
	GetNamespaceByName(ctx context.Context, name string) (*model.Namespace, error)

	// ListNamespaces 获取所有命名空间
	ListNamespaces(ctx context.Context) ([]*model.Namespace, error)

	// DeleteNamespace 删除命名空间
	DeleteNamespace(ctx context.Context, name string) error

	// GetNamespaceServiceCount 获取命名空间下的服务数量
	GetNamespaceServiceCount(ctx context.Context, name string) (int, error)
}
