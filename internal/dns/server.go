package dns

import (
	"context"
	"fmt"
	"log"
	"net"
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

		// 目前仅支持A记录的硬编码响应
		if q.Qtype == dns.TypeA {
			// 简单的硬编码响应，返回127.0.0.1
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
