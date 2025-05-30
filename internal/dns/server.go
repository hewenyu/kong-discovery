package dns

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

// server 实现DNS服务
type server struct {
	config     *Config
	udpServer  *dns.Server
	tcpServer  *dns.Server
	shutdownWg sync.WaitGroup
}

// NewServer 创建一个新的DNS服务实例
func NewServer(config *Config) Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &server{
		config: config,
	}
}

// Start 启动DNS服务器
func (s *server) Start(ctx context.Context) error {
	// 验证必要的配置
	if s.config.ServiceStore == nil {
		log.Println("警告: ServiceStore未设置，将只能返回硬编码响应")
	}

	// 设置DNS处理器
	dnsHandler := dns.NewServeMux()
	dnsHandler.HandleFunc(".", s.handleDNSRequest)

	// 如果启用UDP，启动UDP服务器
	if s.config.EnableUDP {
		s.udpServer = &dns.Server{
			Addr:         s.config.DNSAddr,
			Net:          "udp",
			Handler:      dnsHandler,
			UDPSize:      65535,
			ReadTimeout:  s.config.Timeout,
			WriteTimeout: s.config.Timeout,
		}

		s.shutdownWg.Add(1)
		go func() {
			defer s.shutdownWg.Done()
			log.Printf("启动UDP DNS服务器，监听地址: %s", s.config.DNSAddr)
			if err := s.udpServer.ListenAndServe(); err != nil {
				log.Printf("UDP DNS服务器异常退出: %v", err)
			}
		}()
	}

	// 如果启用TCP，启动TCP服务器
	if s.config.EnableTCP {
		s.tcpServer = &dns.Server{
			Addr:         s.config.DNSAddr,
			Net:          "tcp",
			Handler:      dnsHandler,
			ReadTimeout:  s.config.Timeout,
			WriteTimeout: s.config.Timeout,
		}

		s.shutdownWg.Add(1)
		go func() {
			defer s.shutdownWg.Done()
			log.Printf("启动TCP DNS服务器，监听地址: %s", s.config.DNSAddr)
			if err := s.tcpServer.ListenAndServe(); err != nil {
				log.Printf("TCP DNS服务器异常退出: %v", err)
			}
		}()
	}

	return nil
}

// Stop 停止DNS服务器
func (s *server) Stop() error {
	var errs []error

	// 关闭UDP服务器
	if s.udpServer != nil {
		if err := s.udpServer.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("关闭UDP服务器失败: %w", err))
		}
	}

	// 关闭TCP服务器
	if s.tcpServer != nil {
		if err := s.tcpServer.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("关闭TCP服务器失败: %w", err))
		}
	}

	// 等待所有服务器关闭
	s.shutdownWg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("停止DNS服务器时发生错误: %v", errs)
	}

	return nil
}

// handleDNSRequest 处理DNS请求
func (s *server) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	// 创建响应消息
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	// 处理查询请求
	for _, q := range r.Question {
		log.Printf("收到DNS查询请求: %s %s", q.Name, dns.TypeToString[q.Qtype])

		if q.Qtype == dns.TypeA {
			// 尝试解析服务名和命名空间
			serviceName, namespace, ok := s.parseServiceDomain(q.Name)
			if ok {
				// 如果解析成功，从etcd查询服务实例
				if s.config.ServiceStore != nil {
					m = s.handleServiceLookup(m, q, serviceName, namespace)
					continue
				}
			}

			// 如果不是服务域名或没有配置ServiceStore，返回硬编码响应
			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    s.config.TTL,
				},
				A: net.ParseIP("127.0.0.1"),
			}
			m.Answer = append(m.Answer, rr)
			log.Printf("返回硬编码A记录: %s -> %s", q.Name, "127.0.0.1")
		}
	}

	// 如果没有找到任何记录，返回NXDOMAIN
	if len(m.Answer) == 0 {
		m.Rcode = dns.RcodeNameError
		log.Printf("域名不存在，返回NXDOMAIN")
	}

	// 发送响应
	if err := w.WriteMsg(m); err != nil {
		log.Printf("发送DNS响应失败: %v", err)
	}
}

// parseServiceDomain 解析服务域名
// 格式: service.namespace.domain
// 返回: serviceName, namespace, ok
func (s *server) parseServiceDomain(name string) (string, string, bool) {
	// 移除末尾的点号
	if strings.HasSuffix(name, ".") {
		name = name[:len(name)-1]
	}

	// 检查域名是否使用我们的服务域
	if !strings.HasSuffix(name, s.config.Domain) {
		return "", "", false
	}

	// 移除域名后缀
	name = name[:len(name)-len(s.config.Domain)-1] // 减1是为了移除分隔点

	// 分割服务名和命名空间
	parts := strings.Split(name, ".")
	if len(parts) < 1 {
		return "", "", false
	}

	if len(parts) == 1 {
		// 只有服务名，使用默认命名空间
		return parts[0], "default", true
	}

	// 服务名.命名空间
	return parts[0], parts[1], true
}

// handleServiceLookup 处理服务查询
func (s *server) handleServiceLookup(m *dns.Msg, q dns.Question, serviceName, namespace string) *dns.Msg {
	ctx := context.Background()

	services, err := s.config.ServiceStore.GetServiceByName(ctx, serviceName, namespace)
	if err != nil {
		log.Printf("查询服务[%s.%s]失败: %v", serviceName, namespace, err)
		m.Rcode = dns.RcodeServerFailure
		return m
	}

	if len(services) == 0 {
		log.Printf("服务[%s.%s]不存在", serviceName, namespace)
		m.Rcode = dns.RcodeNameError
		return m
	}

	// 遍历所有健康的服务实例，添加到DNS响应中
	for _, service := range services {
		// 只返回健康的服务实例
		if service.Health != "healthy" {
			continue
		}

		rr := &dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    s.config.TTL,
			},
			A: net.ParseIP(service.IP),
		}

		m.Answer = append(m.Answer, rr)
		log.Printf("返回服务[%s.%s]的A记录: %s", serviceName, namespace, service.IP)
	}

	return m
}
