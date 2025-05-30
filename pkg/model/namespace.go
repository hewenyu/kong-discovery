package model

import "time"

// Namespace 表示一个命名空间
type Namespace struct {
	Name         string    `json:"name"`          // 命名空间名称，唯一标识
	Description  string    `json:"description"`   // 命名空间描述
	CreatedAt    time.Time `json:"created_at"`    // 创建时间
	UpdatedAt    time.Time `json:"updated_at"`    // 更新时间
	ServiceCount int       `json:"service_count"` // 服务数量
}

// DefaultNamespace 默认命名空间名称
const DefaultNamespace = "default"

// NewNamespace 创建一个新的命名空间
func NewNamespace(name, description string) *Namespace {
	now := time.Now()
	return &Namespace{
		Name:         name,
		Description:  description,
		CreatedAt:    now,
		UpdatedAt:    now,
		ServiceCount: 0,
	}
}

// NewDefaultNamespace 创建默认命名空间
func NewDefaultNamespace() *Namespace {
	return NewNamespace(DefaultNamespace, "默认命名空间")
}
