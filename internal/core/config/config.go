package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config 表示应用程序配置
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Etcd      EtcdConfig      `yaml:"etcd"`
	Service   ServiceConfig   `yaml:"service"`
	Namespace NamespaceConfig `yaml:"namespace"`
}

// ServerConfig 表示服务器配置
type ServerConfig struct {
	DNS          DNSConfig          `yaml:"dns"`
	Registration RegistrationConfig `yaml:"registration"`
	Admin        AdminConfig        `yaml:"admin"`
}

// DNSConfig 表示DNS服务配置
type DNSConfig struct {
	Port       int               `yaml:"port"`
	TCPEnabled bool              `yaml:"tcp_enabled"`
	UDPEnabled bool              `yaml:"udp_enabled"`
	Domain     string            `yaml:"domain"`
	TTL        uint32            `yaml:"ttl"`
	Cache      DNSCacheConfig    `yaml:"cache"`
	Upstream   DNSUpstreamConfig `yaml:"upstream"`
}

// DNSCacheConfig 表示DNS缓存配置
type DNSCacheConfig struct {
	Enabled bool  `yaml:"enabled"`
	TTL     int64 `yaml:"ttl"`
}

// DNSUpstreamConfig 表示上游DNS配置
type DNSUpstreamConfig struct {
	Enabled bool     `yaml:"enabled"`
	Servers []string `yaml:"servers"`
}

// RegistrationConfig 表示服务注册API配置
type RegistrationConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// AdminConfig 表示管理API配置
type AdminConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// EtcdConfig 表示etcd配置
type EtcdConfig struct {
	Endpoints      []string      `yaml:"endpoints"`
	DialTimeout    time.Duration `yaml:"dial_timeout"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

// ServiceConfig 表示服务配置
type ServiceConfig struct {
	Heartbeat HeartbeatConfig `yaml:"heartbeat"`
}

// HeartbeatConfig 表示心跳配置
type HeartbeatConfig struct {
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

// NamespaceConfig 表示命名空间配置
type NamespaceConfig struct {
	Default string `yaml:"default"`
}

// LoadConfig 从文件加载配置
func LoadConfig(configFile string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 进行配置验证
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig 验证配置有效性
func validateConfig(config *Config) error {
	// DNS服务配置验证
	if config.Server.DNS.Port <= 0 || config.Server.DNS.Port > 65535 {
		return fmt.Errorf("DNS端口配置无效: %d", config.Server.DNS.Port)
	}
	if !config.Server.DNS.TCPEnabled && !config.Server.DNS.UDPEnabled {
		return fmt.Errorf("DNS服务TCP和UDP不能同时禁用")
	}
	if config.Server.DNS.Domain == "" {
		return fmt.Errorf("DNS域名后缀不能为空")
	}

	// 服务注册API配置验证
	if config.Server.Registration.Port <= 0 || config.Server.Registration.Port > 65535 {
		return fmt.Errorf("服务注册API端口配置无效: %d", config.Server.Registration.Port)
	}

	// 管理API配置验证
	if config.Server.Admin.Port <= 0 || config.Server.Admin.Port > 65535 {
		return fmt.Errorf("管理API端口配置无效: %d", config.Server.Admin.Port)
	}

	// etcd配置验证
	if len(config.Etcd.Endpoints) == 0 {
		return fmt.Errorf("etcd端点不能为空")
	}

	// 心跳配置验证
	if config.Service.Heartbeat.Timeout <= config.Service.Heartbeat.Interval {
		return fmt.Errorf("心跳超时时间必须大于心跳间隔")
	}

	return nil
}
