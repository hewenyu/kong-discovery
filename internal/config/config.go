package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用程序配置结构
type Config struct {
	// etcd配置
	Etcd struct {
		Endpoints []string `mapstructure:"endpoints"`
		Username  string   `mapstructure:"username"`
		Password  string   `mapstructure:"password"`
	} `mapstructure:"etcd"`

	// DNS服务配置
	DNS struct {
		ListenAddress string   `mapstructure:"listen_address"`
		Port          int      `mapstructure:"port"`
		Protocol      string   `mapstructure:"protocol"` // "udp", "tcp", 或 "both"
		UpstreamDNS   []string `mapstructure:"upstream_dns"`
	} `mapstructure:"dns"`

	// API服务配置
	API struct {
		// 管理API端口配置
		Management struct {
			ListenAddress string `mapstructure:"listen_address"`
			Port          int    `mapstructure:"port"`
		} `mapstructure:"management"`

		// 服务注册API端口配置
		Registration struct {
			ListenAddress string `mapstructure:"listen_address"`
			Port          int    `mapstructure:"port"`
		} `mapstructure:"registration"`
	} `mapstructure:"api"`

	// 日志配置
	Log struct {
		Level       string `mapstructure:"level"`
		Development bool   `mapstructure:"development"`
	} `mapstructure:"log"`
}

// LoadConfig 从文件和环境变量加载配置
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 如果指定了配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 设置配置文件名和路径
		v.SetConfigName("config")                // 配置文件名（无扩展名）
		v.AddConfigPath(".")                     // 当前目录
		v.AddConfigPath("./configs")             // configs目录
		v.AddConfigPath("$HOME/.kong-discovery") // 用户目录
		v.AddConfigPath("/etc/kong-discovery")   // 系统目录
	}

	// 配置文件格式
	v.SetConfigType("yaml")

	// 尝试从配置文件加载
	if err := v.ReadInConfig(); err != nil {
		// 如果找不到配置文件，仅记录警告；其他错误则返回
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件错误: %w", err)
		}
	}

	// 绑定环境变量
	v.SetEnvPrefix("KONG_DISCOVERY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 从环境变量覆盖
	bindEnvVariables(v)

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置错误: %w", err)
	}

	return &config, nil
}

// setDefaults 设置配置默认值
func setDefaults(v *viper.Viper) {
	// etcd默认配置
	v.SetDefault("etcd.endpoints", []string{"localhost:2379"})
	v.SetDefault("etcd.username", "")
	v.SetDefault("etcd.password", "")

	// DNS服务默认配置
	v.SetDefault("dns.listen_address", "0.0.0.0")
	v.SetDefault("dns.port", 53)
	v.SetDefault("dns.protocol", "both")
	v.SetDefault("dns.upstream_dns", []string{"8.8.8.8:53", "8.8.4.4:53"})

	// API服务默认配置
	v.SetDefault("api.management.listen_address", "0.0.0.0")
	v.SetDefault("api.management.port", 8080)
	v.SetDefault("api.registration.listen_address", "0.0.0.0")
	v.SetDefault("api.registration.port", 8081)

	// 日志默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.development", true)
}

// bindEnvVariables 绑定特定的环境变量
func bindEnvVariables(v *viper.Viper) {
	// 示例：绑定具体的环境变量
	v.BindEnv("etcd.endpoints", "KONG_DISCOVERY_ETCD_ENDPOINTS")
	v.BindEnv("dns.port", "KONG_DISCOVERY_DNS_PORT")
	v.BindEnv("api.management.port", "KONG_DISCOVERY_MANAGEMENT_API_PORT")
	v.BindEnv("api.registration.port", "KONG_DISCOVERY_REGISTRATION_API_PORT")
}

// GetDefaultConfigPath 返回默认配置文件路径
func GetDefaultConfigPath() string {
	// 按顺序检查不同位置的配置文件
	paths := []string{
		"./config.yaml",
		"./configs/config.yaml",
		os.Getenv("HOME") + "/.kong-discovery/config.yaml",
		"/etc/kong-discovery/config.yaml",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
