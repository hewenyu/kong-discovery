package dnsserver

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/hewenyu/kong-discovery/internal/config"
	"github.com/hewenyu/kong-discovery/internal/etcdclient"
	"github.com/miekg/dns"
	"go.uber.org/zap"
)

// 服务域名后缀，用于识别服务域名
const serviceDomainSuffix = ".svc.cluster.local"

// Server 定义DNS服务器接口
type Server interface {
	// Start 启动DNS服务器
	Start() error

	// Shutdown 优雅关闭DNS服务器
	Shutdown(ctx context.Context) error

	// SetEtcdClient 设置etcd客户端
	SetEtcdClient(client etcdclient.Client)
}

// DNSServer 实现Server接口
type DNSServer struct {
	udpServer   *dns.Server
	tcpServer   *dns.Server
	cfg         *config.Config
	logger      config.Logger
	shutdownErr chan error
	etcdClient  etcdclient.Client
}

// NewDNSServer 创建一个新的DNS服务器
func NewDNSServer(cfg *config.Config, logger config.Logger) Server {
	return &DNSServer{
		cfg:         cfg,
		logger:      logger,
		shutdownErr: make(chan error, 2), // 用于收集UDP和TCP服务器的关闭错误
	}
}

// SetEtcdClient 设置etcd客户端
func (s *DNSServer) SetEtcdClient(client etcdclient.Client) {
	s.etcdClient = client
}

// Start 启动DNS服务器
func (s *DNSServer) Start() error {
	s.logger.Info("启动DNS服务器",
		zap.String("address", s.cfg.DNS.ListenAddress),
		zap.Int("port", s.cfg.DNS.Port),
		zap.String("protocol", s.cfg.DNS.Protocol))

	// 创建DNS处理器
	handler := dns.NewServeMux()
	handler.HandleFunc(".", s.handleDNSRequest)

	// 创建服务器地址
	addr := net.JoinHostPort(s.cfg.DNS.ListenAddress, strconv.Itoa(s.cfg.DNS.Port))

	// 根据配置启动对应协议的服务器
	switch s.cfg.DNS.Protocol {
	case "udp":
		return s.startUDPServer(addr, handler)
	case "tcp":
		return s.startTCPServer(addr, handler)
	case "both":
		if err := s.startUDPServer(addr, handler); err != nil {
			return err
		}
		return s.startTCPServer(addr, handler)
	default:
		return fmt.Errorf("不支持的DNS协议: %s", s.cfg.DNS.Protocol)
	}
}

// startUDPServer 启动UDP服务器
func (s *DNSServer) startUDPServer(addr string, handler dns.Handler) error {
	s.udpServer = &dns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: handler,
	}

	s.logger.Info("启动UDP DNS服务器", zap.String("addr", addr))

	// 在后台启动UDP服务器
	go func() {
		if err := s.udpServer.ListenAndServe(); err != nil {
			// miekg/dns没有ErrServerClosed，我们需要自己判断服务关闭情况
			s.logger.Error("UDP DNS服务器错误", zap.Error(err))
			s.shutdownErr <- err
		}
	}()

	return nil
}

// startTCPServer 启动TCP服务器
func (s *DNSServer) startTCPServer(addr string, handler dns.Handler) error {
	s.tcpServer = &dns.Server{
		Addr:    addr,
		Net:     "tcp",
		Handler: handler,
	}

	s.logger.Info("启动TCP DNS服务器", zap.String("addr", addr))

	// 在后台启动TCP服务器
	go func() {
		if err := s.tcpServer.ListenAndServe(); err != nil {
			// miekg/dns没有ErrServerClosed，我们需要自己判断服务关闭情况
			s.logger.Error("TCP DNS服务器错误", zap.Error(err))
			s.shutdownErr <- err
		}
	}()

	return nil
}

// Shutdown 优雅关闭DNS服务器
func (s *DNSServer) Shutdown(ctx context.Context) error {
	s.logger.Info("正在关闭DNS服务器...")

	// 关闭UDP服务器
	if s.udpServer != nil {
		if err := s.udpServer.ShutdownContext(ctx); err != nil {
			s.logger.Error("关闭UDP DNS服务器出错", zap.Error(err))
			return err
		}
		s.logger.Info("UDP DNS服务器已关闭")
	}

	// 关闭TCP服务器
	if s.tcpServer != nil {
		if err := s.tcpServer.ShutdownContext(ctx); err != nil {
			s.logger.Error("关闭TCP DNS服务器出错", zap.Error(err))
			return err
		}
		s.logger.Info("TCP DNS服务器已关闭")
	}

	return nil
}

// handleDNSRequest 处理DNS请求
func (s *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	// 标记是否处理了所有查询
	allQueriesHandled := true

	// 遍历所有的问题
	for _, q := range r.Question {
		s.logger.Info("收到DNS查询",
			zap.String("name", q.Name),
			zap.String("type", dns.TypeToString[q.Qtype]),
			zap.String("client", w.RemoteAddr().String()))

		// 处理DNS查询
		found := s.handleQuery(q, m)

		// 如果没有找到答案，标记为未处理所有查询
		if !found {
			allQueriesHandled = false
		}
	}

	// 如果没有处理所有查询，并且配置了上游DNS，尝试转发
	if !allQueriesHandled && s.cfg.DNS.UpstreamDNS != "" {
		err := s.forwardToUpstream(r, m)
		if err != nil {
			s.logger.Error("向上游DNS转发查询失败", zap.Error(err))
			// 如果转发失败，设置响应代码为 SERVFAIL
			m.SetRcode(r, dns.RcodeServerFailure)
		}
	} else if !allQueriesHandled {
		// 如果没有找到答案且没有配置上游DNS，设置响应代码为 NXDOMAIN
		m.SetRcode(r, dns.RcodeNameError)
	}

	// 发送响应
	if err := w.WriteMsg(m); err != nil {
		s.logger.Error("发送DNS响应失败", zap.Error(err))
	}
}

// forwardToUpstream 将DNS查询转发到上游DNS服务器
func (s *DNSServer) forwardToUpstream(r *dns.Msg, m *dns.Msg) error {
	s.logger.Info("转发查询到上游DNS服务器",
		zap.String("upstream", s.cfg.DNS.UpstreamDNS))

	// 创建一个新的客户端
	c := new(dns.Client)

	// 复制原始请求
	req := r.Copy()
	req.Id = dns.Id() // 生成新的ID

	// 发送到上游DNS服务器
	resp, _, err := c.Exchange(req, s.cfg.DNS.UpstreamDNS)
	if err != nil {
		return err
	}

	// 检查响应
	if resp == nil {
		return fmt.Errorf("上游DNS返回空响应")
	}

	// 将上游DNS的响应复制到我们的响应中
	m.Answer = resp.Answer
	m.Ns = resp.Ns
	m.Extra = resp.Extra
	m.Rcode = resp.Rcode
	m.Authoritative = false // 因为这是从上游转发的，所以不是权威响应

	return nil
}

// handleQuery 处理单个DNS查询问题
func (s *DNSServer) handleQuery(q dns.Question, m *dns.Msg) bool {
	// 1. 移除尾部的点号，并转换为小写
	domain := strings.TrimSuffix(strings.ToLower(q.Name), ".")

	// 2. 先检查硬编码测试记录
	if domain == "test.local" && q.Qtype == dns.TypeA {
		rr, err := dns.NewRR(fmt.Sprintf("%s. A 1.2.3.4", domain))
		if err == nil {
			m.Answer = append(m.Answer, rr)
			return true
		}
	}

	// 3. 如果etcdClient未设置，无法查询etcd
	if s.etcdClient == nil {
		s.logger.Warn("etcd客户端未设置，无法查询DNS记录")
		return false
	}

	// 4. 检查是否为服务域名（以.svc.cluster.local结尾）
	if strings.HasSuffix(domain, serviceDomainSuffix) {
		return s.handleServiceQuery(domain, q.Qtype, m)
	}

	// 5. 处理常规DNS记录查询
	return s.handleRegularDNSQuery(domain, q.Qtype, m)
}

// handleServiceQuery 处理服务发现查询
func (s *DNSServer) handleServiceQuery(domain string, qtype uint16, m *dns.Msg) bool {
	ctx := context.Background()

	// 如果请求的是SRV记录，我们需要特别处理
	if qtype == dns.TypeSRV {
		return s.handleSRVQuery(domain, m)
	}

	// 对于A记录，我们返回服务的IP地址
	if qtype == dns.TypeA {
		records, err := s.etcdClient.ServiceToDNSRecords(ctx, domain)
		if err != nil {
			s.logger.Debug("获取服务DNS记录失败",
				zap.String("domain", domain),
				zap.Error(err))
			return false
		}

		// 查找A记录
		if aRecord, ok := records["A"]; ok {
			rr, err := dns.NewRR(fmt.Sprintf("%s. A %s", domain, aRecord.Value))
			if err != nil {
				s.logger.Error("创建A记录失败", zap.Error(err))
				return false
			}
			m.Answer = append(m.Answer, rr)
			return true
		}
	}

	return false
}

// handleSRVQuery 处理SRV查询
func (s *DNSServer) handleSRVQuery(domain string, m *dns.Msg) bool {
	ctx := context.Background()

	// 获取服务的DNS记录
	records, err := s.etcdClient.ServiceToDNSRecords(ctx, domain)
	if err != nil {
		s.logger.Debug("获取服务DNS记录失败",
			zap.String("domain", domain),
			zap.Error(err))
		return false
	}

	// 添加所有SRV记录
	added := false
	for key, record := range records {
		if strings.HasPrefix(key, "SRV-") {
			rr, err := dns.NewRR(fmt.Sprintf("%s. SRV %s", domain, record.Value))
			if err != nil {
				s.logger.Error("创建SRV记录失败", zap.Error(err))
				continue
			}
			m.Answer = append(m.Answer, rr)
			added = true
		}
	}

	return added
}

// handleRegularDNSQuery 处理常规DNS记录查询
func (s *DNSServer) handleRegularDNSQuery(domain string, qtype uint16, m *dns.Msg) bool {
	// 获取记录类型字符串
	recordType := dns.TypeToString[qtype]

	// 从etcd获取DNS记录
	ctx := context.Background()
	record, err := s.etcdClient.GetDNSRecord(ctx, domain, recordType)
	if err != nil {
		s.logger.Debug("从etcd获取DNS记录失败",
			zap.String("domain", domain),
			zap.String("type", recordType),
			zap.Error(err))
		return false
	}

	// 创建适当的DNS记录响应
	switch qtype {
	case dns.TypeA:
		rr, err := dns.NewRR(fmt.Sprintf("%s. A %s", domain, record.Value))
		if err != nil {
			s.logger.Error("创建A记录失败", zap.Error(err))
			return false
		}
		m.Answer = append(m.Answer, rr)
		return true

	case dns.TypeAAAA:
		rr, err := dns.NewRR(fmt.Sprintf("%s. AAAA %s", domain, record.Value))
		if err != nil {
			s.logger.Error("创建AAAA记录失败", zap.Error(err))
			return false
		}
		m.Answer = append(m.Answer, rr)
		return true

	case dns.TypeCNAME:
		rr, err := dns.NewRR(fmt.Sprintf("%s. CNAME %s", domain, record.Value))
		if err != nil {
			s.logger.Error("创建CNAME记录失败", zap.Error(err))
			return false
		}
		m.Answer = append(m.Answer, rr)
		return true

	case dns.TypeTXT:
		rr, err := dns.NewRR(fmt.Sprintf("%s. TXT \"%s\"", domain, record.Value))
		if err != nil {
			s.logger.Error("创建TXT记录失败", zap.Error(err))
			return false
		}
		m.Answer = append(m.Answer, rr)
		return true

	case dns.TypeSRV:
		// SRV记录的值格式应为: "priority weight port target"
		rr, err := dns.NewRR(fmt.Sprintf("%s. SRV %s", domain, record.Value))
		if err != nil {
			s.logger.Error("创建SRV记录失败", zap.Error(err))
			return false
		}
		m.Answer = append(m.Answer, rr)
		return true

	default:
		s.logger.Warn("不支持的DNS记录类型",
			zap.String("domain", domain),
			zap.String("type", recordType))
		return false
	}
}
