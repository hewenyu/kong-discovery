package dns

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/hewenyu/kong-discovery/internal/core/model"
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

	// 记录是否找到了本地服务记录
	foundLocalRecord := false

	// 处理查询请求
	for _, q := range r.Question {
		log.Printf("收到DNS查询请求: %s %s", q.Name, dns.TypeToString[q.Qtype])

		// 尝试解析服务名和命名空间
		serviceName, namespace, ok := s.parseServiceDomain(q.Name)
		if ok {
			// 如果解析成功，从etcd查询服务实例
			if s.config.ServiceStore != nil {
				switch q.Qtype {
				case dns.TypeA:
					// A记录查询
					m = s.handleARecordLookup(m, q, serviceName, namespace)
					if len(m.Answer) > 0 {
						foundLocalRecord = true
					}
					continue
				case dns.TypeSRV:
					// SRV记录查询
					m = s.handleSRVRecordLookup(m, q, serviceName, namespace)
					if len(m.Answer) > 0 {
						foundLocalRecord = true
					}
					continue
				}
			}
		}

		// 对于非服务域名或不支持的查询类型，我们不再返回硬编码响应
		// 删除旧的硬编码A记录逻辑
	}

	// 如果没有找到任何本地记录，并且配置了上游DNS，则转发到上游DNS
	if !foundLocalRecord && len(m.Answer) == 0 {
		if len(s.config.UpstreamDNS) > 0 {
			log.Printf("本地无匹配记录，转发到上游DNS")
			resp, err := s.forwardToUpstream(r)
			if err != nil {
				log.Printf("转发到上游DNS失败: %v", err)
				m.Rcode = dns.RcodeServerFailure
			} else {
				// 使用上游返回的响应替换我们的响应
				*m = *resp
			}
		} else {
			// 如果没有配置上游DNS，返回NXDOMAIN
			m.Rcode = dns.RcodeNameError
			log.Printf("域名不存在且未配置上游DNS，返回NXDOMAIN")
		}
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
	name = strings.TrimSuffix(name, ".")

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

// filterHealthyServices 只过滤出健康的服务，不进行轮询排序
func (s *server) filterHealthyServices(services []*model.Service) []*model.Service {
	// 筛选出健康的服务
	var healthyServices []*model.Service
	for _, svc := range services {
		if svc.Health == "healthy" {
			healthyServices = append(healthyServices, svc)
		}
	}

	return healthyServices
}

// handleARecordLookup 处理A记录查询
func (s *server) handleARecordLookup(m *dns.Msg, q dns.Question, serviceName, namespace string) *dns.Msg {
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

	// 只过滤健康服务，不进行轮询
	healthyServices := s.filterHealthyServices(services)
	if len(healthyServices) == 0 {
		log.Printf("服务[%s.%s]没有健康的实例", serviceName, namespace)
		m.Rcode = dns.RcodeNameError
		return m
	}

	// 遍历所有健康的服务实例，添加到DNS响应中
	for _, service := range healthyServices {
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

// handleSRVRecordLookup 处理SRV记录查询
func (s *server) handleSRVRecordLookup(m *dns.Msg, q dns.Question, serviceName, namespace string) *dns.Msg {
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

	// 只过滤健康服务，不进行轮询
	healthyServices := s.filterHealthyServices(services)
	if len(healthyServices) == 0 {
		log.Printf("服务[%s.%s]没有健康的实例", serviceName, namespace)
		m.Rcode = dns.RcodeNameError
		return m
	}

	// 遍历所有健康的服务实例，添加到DNS响应中
	for idx, service := range healthyServices {
		// 为每个服务实例创建一个唯一的目标名称
		// 格式：instance-{idx}.{service}.{namespace}.{domain}
		targetDomain := fmt.Sprintf("instance-%d.%s.%s.%s.", idx, serviceName, namespace, s.config.Domain)

		// 创建SRV记录
		// SRV记录优先级默认为0（最高），权重设为0，不使用权重进行负载均衡
		srvRR := &dns.SRV{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeSRV,
				Class:  dns.ClassINET,
				Ttl:    s.config.TTL,
			},
			Priority: 0, // 优先级，0为最高
			Weight:   0, // 权重设为0，由网关自行处理负载均衡
			Port:     uint16(service.Port),
			Target:   targetDomain,
		}

		// 创建相应的A记录，解析SRV记录的目标域名
		aRR := &dns.A{
			Hdr: dns.RR_Header{
				Name:   targetDomain,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    s.config.TTL,
			},
			A: net.ParseIP(service.IP),
		}

		// 将SRV记录添加到回答中
		m.Answer = append(m.Answer, srvRR)

		// 将A记录添加到附加部分
		m.Extra = append(m.Extra, aRR)

		log.Printf("返回服务[%s.%s]的SRV记录: %s:%d", serviceName, namespace, service.IP, service.Port)
	}

	return m
}

// forwardToUpstream 将DNS请求转发到上游DNS服务器
func (s *server) forwardToUpstream(r *dns.Msg) (*dns.Msg, error) {
	// 创建一个新的DNS客户端
	c := new(dns.Client)
	c.Timeout = s.config.Timeout

	// 尝试每个上游DNS服务器，直到成功或全部失败
	var lastErr error
	for _, upstreamAddr := range s.config.UpstreamDNS {
		log.Printf("转发DNS查询到上游服务器: %s", upstreamAddr)

		// 发送请求到上游DNS服务器
		resp, _, err := c.Exchange(r, upstreamAddr)
		if err != nil {
			log.Printf("上游DNS服务器 %s 请求失败: %v", upstreamAddr, err)
			lastErr = err
			continue
		}

		// 如果响应是截断的(TC=1)，可以尝试使用TCP重新查询
		if resp.Truncated {
			log.Printf("上游DNS响应被截断，尝试使用TCP重新查询")
			c.Net = "tcp"
			resp, _, err = c.Exchange(r, upstreamAddr)
			if err != nil {
				log.Printf("上游DNS服务器 %s TCP请求失败: %v", upstreamAddr, err)
				lastErr = err
				continue
			}
		}

		log.Printf("从上游DNS服务器 %s 成功获取响应", upstreamAddr)
		return resp, nil
	}

	// 所有上游DNS服务器都失败了
	return nil, fmt.Errorf("所有上游DNS服务器都失败: %v", lastErr)
}
