package dns

import (
	"context"
	"time"
)

// Service 定义DNS服务的接口
type Service interface {
	// Start 启动DNS服务
	Start(ctx context.Context) error

	// Stop 停止DNS服务
	Stop() error
}

// Config 定义DNS服务的配置项
type Config struct {
	// DNSAddr 是DNS服务的监听地址，格式为 "ip:port"
	DNSAddr string

	// Domain 是服务域名后缀
	Domain string

	// TTL 是DNS响应的存活时间
	TTL uint32

	// Timeout 是DNS查询的超时时间
	Timeout time.Duration

	// UpstreamDNS 是上游DNS服务器地址列表
	UpstreamDNS []string

	// EnableTCP 是否启用TCP监听
	EnableTCP bool

	// EnableUDP 是否启用UDP监听
	EnableUDP bool
}

// DefaultConfig 返回默认的DNS服务配置
func DefaultConfig() *Config {
	return &Config{
		DNSAddr:     ":53",
		Domain:      "service.local",
		TTL:         60,
		Timeout:     5 * time.Second,
		UpstreamDNS: []string{"8.8.8.8:53", "114.114.114.114:53"},
		EnableTCP:   true,
		EnableUDP:   true,
	}
}
