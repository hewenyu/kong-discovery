package dns

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/miekg/dns"

	"github.com/hewenyu/kong-discovery/pkg/config"
	"github.com/hewenyu/kong-discovery/pkg/storage"
)

// Server DNS服务器
type Server struct {
	udpServer        *dns.Server         // UDP服务器
	tcpServer        *dns.Server         // TCP服务器
	handler          *Handler            // DNS请求处理器
	recordManager    *RecordManager      // DNS记录管理器
	upstreamResolver *UpstreamResolver   // 上游DNS解析器
	cache            *DNSCache           // DNS缓存
	config           config.DNSConfig    // DNS配置
	serverConfig     config.ServerConfig // 服务器配置
	refreshTicker    *time.Ticker        // 定时刷新计时器
	cancelFunc       context.CancelFunc  // 用于停止服务器的函数
}

// NewServer 创建DNS服务器
func NewServer(conf *config.Config, storage storage.ServiceStorage) (*Server, error) {
	// 创建DNS缓存
	cache := NewDNSCache(conf.DNS.CacheTTL)

	// 创建记录管理器
	recordManager := NewRecordManager(storage, conf.DNS.Domain, conf.DNS.CacheTTL)

	// 创建上游解析器
	upstreamResolver := NewUpstreamResolver(conf.DNS.Upstream, cache)

	// 创建DNS处理器
	handler := NewHandler(recordManager, upstreamResolver, cache, conf.DNS.Domain)

	// 创建服务器
	server := &Server{
		udpServer: &dns.Server{
			Addr:    fmt.Sprintf(":%d", conf.Server.DNSPort),
			Net:     "udp",
			Handler: handler,
		},
		tcpServer: &dns.Server{
			Addr:    fmt.Sprintf(":%d", conf.Server.DNSPort),
			Net:     "tcp",
			Handler: handler,
		},
		handler:          handler,
		recordManager:    recordManager,
		upstreamResolver: upstreamResolver,
		cache:            cache,
		config:           conf.DNS,
		serverConfig:     conf.Server,
	}

	return server, nil
}

// Start 启动DNS服务器
func (s *Server) Start(ctx context.Context) error {
	// 创建带取消功能的上下文
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	// 初始化DNS记录
	if err := s.recordManager.RefreshRecords(ctx); err != nil {
		return fmt.Errorf("刷新DNS记录失败: %w", err)
	}

	// 启动定期刷新DNS记录
	s.refreshTicker = time.NewTicker(time.Duration(s.config.CacheTTL) * time.Second)
	go func() {
		for {
			select {
			case <-s.refreshTicker.C:
				if err := s.recordManager.RefreshRecords(ctx); err != nil {
					log.Printf("刷新DNS记录失败: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 启动定期清理缓存
	s.cache.StartCleanupRoutine(time.Minute)

	// 启动UDP服务器
	go func() {
		log.Printf("DNS UDP服务器启动，监听端口 %d", s.serverConfig.DNSPort)
		if err := s.udpServer.ListenAndServe(); err != nil {
			log.Printf("DNS UDP服务器启动失败: %v", err)
		}
	}()

	// 启动TCP服务器
	go func() {
		log.Printf("DNS TCP服务器启动，监听端口 %d", s.serverConfig.DNSPort)
		if err := s.tcpServer.ListenAndServe(); err != nil {
			log.Printf("DNS TCP服务器启动失败: %v", err)
		}
	}()

	return nil
}

// Stop 停止DNS服务器
func (s *Server) Stop() error {
	// 取消后台任务
	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	// 停止定时器
	if s.refreshTicker != nil {
		s.refreshTicker.Stop()
	}

	// 关闭UDP服务器
	if err := s.udpServer.Shutdown(); err != nil {
		log.Printf("关闭DNS UDP服务器失败: %v", err)
	}

	// 关闭TCP服务器
	if err := s.tcpServer.Shutdown(); err != nil {
		log.Printf("关闭DNS TCP服务器失败: %v", err)
	}

	return nil
}
