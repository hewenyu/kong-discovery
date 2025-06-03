package model

import (
	"time"
)

// Namespace 表示一个命名空间
type Namespace struct {
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ServiceCount int       `json:"service_count"`
}

// NamespaceCreateRequest 表示创建命名空间的请求
type NamespaceCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
}

// NamespaceResponse 表示命名空间响应
type NamespaceResponse struct {
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ServiceCount int       `json:"service_count"`
}

// NamespaceListResponse 表示命名空间列表响应
type NamespaceListResponse struct {
	Namespaces []*Namespace `json:"namespaces"`
}
