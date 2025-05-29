package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 定义整个应用的配置结构
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Etcd      EtcdConfig      `mapstructure:"etcd"`
	DNS       DNSConfig       `mapstructure:"dns"`
	Heartbeat HeartbeatConfig `mapstructure:"heartbeat"`
	Log       LogConfig       `mapstructure:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	RegisterPort int `mapstructure:"register_port"`
	AdminPort    int `mapstructure:"admin_port"`
	DNSPort      int `mapstructure:"dns_port"`
}

// EtcdConfig etcd配置
type EtcdConfig struct {
	Endpoints   []string `mapstructure:"endpoints"`
	DialTimeout string   `mapstructure:"dial_timeout"`
	Username    string   `mapstructure:"username"`
	Password    string   `mapstructure:"password"`
}

// DNSConfig DNS服务配置
type DNSConfig struct {
	Domain   string   `mapstructure:"domain"`
	Upstream []string `mapstructure:"upstream"`
	CacheTTL int      `mapstructure:"cache_ttl"`
}

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	Interval int `mapstructure:"interval"`
	Timeout  int `mapstructure:"timeout"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// LoadConfig 从文件和环境变量加载配置
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 设置配置文件
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 默认查找路径
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/kong-discovery")
		v.SetConfigName("config")
	}
	v.SetConfigType("yaml")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 配置文件不存在时不返回错误
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件错误: %w", err)
		}
		fmt.Println("未找到配置文件，使用默认配置和环境变量")
	}

	// 从环境变量读取配置
	v.SetEnvPrefix("KONG_DISCOVERY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 解析配置到结构体
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置错误: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// 服务器默认配置
	v.SetDefault("server.register_port", 8080)
	v.SetDefault("server.admin_port", 9090)
	v.SetDefault("server.dns_port", 53)

	// etcd默认配置
	v.SetDefault("etcd.endpoints", []string{"localhost:2379"})
	v.SetDefault("etcd.dial_timeout", "5s")

	// DNS默认配置
	v.SetDefault("dns.domain", "service.local")
	v.SetDefault("dns.upstream", []string{"8.8.8.8:53", "114.114.114.114:53"})
	v.SetDefault("dns.cache_ttl", 60)

	// 心跳默认配置
	v.SetDefault("heartbeat.interval", 30)
	v.SetDefault("heartbeat.timeout", 90)

	// 日志默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.file", "logs/kong-discovery.log")
}
