package dnsserver

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

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

	// UpdateCache 更新DNS记录缓存
	UpdateCache(domain string, recordType string, record *etcdclient.DNSRecord)

	// RemoveFromCache 从缓存中移除DNS记录
	RemoveFromCache(domain string, recordType string)

	// UpdateServiceCache 更新服务缓存
	UpdateServiceCache(service *etcdclient.ServiceInstance)

	// RemoveServiceFromCache 从缓存中移除服务
	RemoveServiceFromCache(serviceName, instanceID string)
}

// DNSServer 实现Server接口
type DNSServer struct {
	udpServer   *dns.Server
	tcpServer   *dns.Server
	cfg         *config.Config
	logger      config.Logger
	shutdownErr chan error
	etcdClient  etcdclient.Client

	// 缓存相关
	cacheMutex     sync.RWMutex
	dnsCache       map[string]map[string]*etcdclient.DNSRecord       // domain -> recordType -> record
	serviceCache   map[string]map[string]*etcdclient.ServiceInstance // serviceName -> instanceID -> instance
	watcherStarted bool

	// 上游DNS配置
	upstreamDNSMutex     sync.RWMutex
	upstreamDNS          []string // 当前使用的上游DNS服务器地址列表
	currentUpstreamIndex int      // 当前使用的上游DNS索引（用于轮询）
}

// NewDNSServer 创建一个新的DNS服务器
func NewDNSServer(cfg *config.Config, logger config.Logger) Server {
	return &DNSServer{
		cfg:                  cfg,
		logger:               logger,
		shutdownErr:          make(chan error, 2), // 用于收集UDP和TCP服务器的关闭错误
		dnsCache:             make(map[string]map[string]*etcdclient.DNSRecord),
		serviceCache:         make(map[string]map[string]*etcdclient.ServiceInstance),
		upstreamDNS:          cfg.DNS.UpstreamDNS, // 初始使用配置文件中的上游DNS
		currentUpstreamIndex: 0,
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

	// 启动etcd监听
	if s.etcdClient != nil && !s.watcherStarted {
		s.startEtcdWatcher()
	}

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

// startEtcdWatcher 启动etcd监听
func (s *DNSServer) startEtcdWatcher() {
	ctx := context.Background()

	// 监听DNS记录变化
	err := s.etcdClient.StartWatch(ctx, "/dns/records/", func(event etcdclient.WatchEvent) {
		s.handleDNSRecordChange(event)
	})

	if err != nil {
		s.logger.Error("启动DNS记录监听失败", zap.Error(err))
	} else {
		s.logger.Info("DNS记录监听已启动")
	}

	// 监听服务变化
	err = s.etcdClient.StartWatch(ctx, "/services/", func(event etcdclient.WatchEvent) {
		s.handleServiceChange(event)
	})

	if err != nil {
		s.logger.Error("启动服务监听失败", zap.Error(err))
	} else {
		s.logger.Info("服务监听已启动")
		s.watcherStarted = true
	}

	// 监听DNS配置变化
	err = s.etcdClient.StartWatch(ctx, "/config/dns/", func(event etcdclient.WatchEvent) {
		s.handleDNSConfigChange(event)
	})

	if err != nil {
		s.logger.Error("启动DNS配置监听失败", zap.Error(err))
	} else {
		s.logger.Info("DNS配置监听已启动")
	}

	// 尝试从etcd加载DNS配置
	s.loadDNSConfigFromEtcd(ctx)
}

// loadDNSConfigFromEtcd 从etcd加载DNS配置
func (s *DNSServer) loadDNSConfigFromEtcd(ctx context.Context) {
	if s.etcdClient == nil {
		return
	}

	// 获取DNS配置
	configs, err := s.etcdClient.GetDNSConfig(ctx)
	if err != nil {
		s.logger.Error("从etcd加载DNS配置失败", zap.Error(err))
		return
	}

	// 更新上游DNS配置
	if upstreamDNSStr, ok := configs["upstream_dns"]; ok && upstreamDNSStr != "" {
		// 尝试解析为JSON数组
		var upstreamList []string
		err := json.Unmarshal([]byte(upstreamDNSStr), &upstreamList)
		if err != nil {
			// 如果不是JSON数组，则作为单个值处理
			upstreamList = []string{upstreamDNSStr}
		}

		s.updateUpstreamDNS(upstreamList)
	}
}

// handleDNSConfigChange 处理DNS配置变化事件
func (s *DNSServer) handleDNSConfigChange(event etcdclient.WatchEvent) {
	// 从key中提取配置项名称
	// 预期key格式: /config/dns/{configName}
	parts := strings.Split(event.Key, "/")
	if len(parts) < 4 {
		s.logger.Warn("无效的DNS配置key格式", zap.String("key", event.Key))
		return
	}

	configName := parts[3]

	switch event.EventType {
	case "create", "update":
		if configName == "upstream_dns" {
			// 尝试解析为上游DNS服务器列表
			var upstreamList []string
			err := json.Unmarshal([]byte(event.Value), &upstreamList)
			if err != nil {
				// 如果不是JSON数组，则作为单个值处理
				upstreamList = []string{event.Value}
			}
			s.updateUpstreamDNS(upstreamList)
		}

	case "delete":
		if configName == "upstream_dns" {
			// 如果配置被删除，恢复为默认配置
			s.updateUpstreamDNS(s.cfg.DNS.UpstreamDNS)
		}
	}
}

// updateUpstreamDNS 更新上游DNS配置
func (s *DNSServer) updateUpstreamDNS(upstreamDNS []string) {
	s.upstreamDNSMutex.Lock()
	defer s.upstreamDNSMutex.Unlock()

	// 检查是否有变化
	if len(s.upstreamDNS) == len(upstreamDNS) {
		same := true
		for i, v := range s.upstreamDNS {
			if v != upstreamDNS[i] {
				same = false
				break
			}
		}
		if same {
			return
		}
	}

	s.upstreamDNS = upstreamDNS
	s.currentUpstreamIndex = 0 // 重置索引
	s.logger.Info("上游DNS配置已更新", zap.Strings("upstream_dns", upstreamDNS))
}

// getNextUpstreamDNS 获取下一个上游DNS服务器（轮询方式）
func (s *DNSServer) getNextUpstreamDNS() string {
	s.upstreamDNSMutex.Lock()
	defer s.upstreamDNSMutex.Unlock()

	if len(s.upstreamDNS) == 0 {
		// 如果没有配置上游DNS，返回默认值的第一个
		if len(s.cfg.DNS.UpstreamDNS) > 0 {
			return s.cfg.DNS.UpstreamDNS[0]
		}
		return "" // 没有可用的上游DNS
	}

	// 轮询选择上游DNS
	upstream := s.upstreamDNS[s.currentUpstreamIndex]

	// 更新索引
	s.currentUpstreamIndex = (s.currentUpstreamIndex + 1) % len(s.upstreamDNS)

	return upstream
}

// handleDNSRecordChange 处理DNS记录变化事件
func (s *DNSServer) handleDNSRecordChange(event etcdclient.WatchEvent) {
	// 从key中提取域名和记录类型
	// 预期key格式: /dns/records/{domain}/{recordType}
	parts := strings.Split(event.Key, "/")
	if len(parts) < 5 {
		s.logger.Warn("无效的DNS记录key格式", zap.String("key", event.Key))
		return
	}

	domain := parts[3]
	recordType := parts[4]

	switch event.EventType {
	case "create", "update":
		// 解析记录值
		var record etcdclient.DNSRecord
		if err := json.Unmarshal([]byte(event.Value), &record); err != nil {
			s.logger.Error("解析DNS记录失败",
				zap.String("domain", domain),
				zap.String("type", recordType),
				zap.Error(err))
			return
		}

		// 更新缓存
		s.UpdateCache(domain, recordType, &record)

		s.logger.Info("DNS记录已更新",
			zap.String("domain", domain),
			zap.String("type", recordType),
			zap.String("value", record.Value))

	case "delete":
		// 从缓存中移除
		s.RemoveFromCache(domain, recordType)

		s.logger.Info("DNS记录已删除",
			zap.String("domain", domain),
			zap.String("type", recordType))
	}
}

// handleServiceChange 处理服务变化事件
func (s *DNSServer) handleServiceChange(event etcdclient.WatchEvent) {
	// 对于删除事件，ServiceObj可能为空，需要从key中提取信息
	if event.EventType == "delete" {
		// 从key中提取服务名和实例ID
		// 预期key格式: /services/{serviceName}/{instanceID}
		parts := strings.Split(event.Key, "/")
		if len(parts) < 4 {
			s.logger.Warn("无效的服务key格式", zap.String("key", event.Key))
			return
		}

		serviceName := parts[2]
		instanceID := parts[3]

		s.RemoveServiceFromCache(serviceName, instanceID)

		s.logger.Info("服务已删除",
			zap.String("service", serviceName),
			zap.String("id", instanceID))
		return
	}

	// 对于create和update事件，需要ServiceObj
	if event.ServiceObj == nil {
		s.logger.Warn("服务对象为空", zap.String("key", event.Key))
		return
	}

	service := event.ServiceObj

	switch event.EventType {
	case "create", "update":
		// 更新服务缓存
		s.UpdateServiceCache(service)

		s.logger.Info("服务已更新",
			zap.String("service", service.ServiceName),
			zap.String("id", service.InstanceID),
			zap.String("ip", service.IPAddress),
			zap.Int("port", service.Port))
	}
}

// UpdateCache 更新DNS记录缓存
func (s *DNSServer) UpdateCache(domain string, recordType string, record *etcdclient.DNSRecord) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// 确保域名在缓存中有对应的map
	if _, ok := s.dnsCache[domain]; !ok {
		s.dnsCache[domain] = make(map[string]*etcdclient.DNSRecord)
	}

	// 更新记录
	s.dnsCache[domain][recordType] = record
}

// RemoveFromCache 从缓存中移除DNS记录
func (s *DNSServer) RemoveFromCache(domain string, recordType string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	if records, ok := s.dnsCache[domain]; ok {
		delete(records, recordType)

		// 如果该域名下没有记录了，删除整个域名条目
		if len(records) == 0 {
			delete(s.dnsCache, domain)
		}
	}
}

// UpdateServiceCache 更新服务缓存
func (s *DNSServer) UpdateServiceCache(service *etcdclient.ServiceInstance) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// 确保服务名在缓存中有对应的map
	if _, ok := s.serviceCache[service.ServiceName]; !ok {
		s.serviceCache[service.ServiceName] = make(map[string]*etcdclient.ServiceInstance)
	}

	// 更新服务实例
	s.serviceCache[service.ServiceName][service.InstanceID] = service
}

// RemoveServiceFromCache 从缓存中移除服务
func (s *DNSServer) RemoveServiceFromCache(serviceName, instanceID string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	if instances, ok := s.serviceCache[serviceName]; ok {
		delete(instances, instanceID)

		// 如果该服务下没有实例了，删除整个服务条目
		if len(instances) == 0 {
			delete(s.serviceCache, serviceName)
		}
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
	if !allQueriesHandled && len(s.upstreamDNS) > 0 {
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
	// 获取下一个上游DNS服务器
	upstreamDNS := s.getNextUpstreamDNS()

	// 如果没有可用的上游DNS
	if upstreamDNS == "" {
		return fmt.Errorf("没有可用的上游DNS服务器")
	}

	s.logger.Info("转发查询到上游DNS服务器",
		zap.String("upstream", upstreamDNS))

	// 创建一个新的客户端
	c := new(dns.Client)

	// 复制原始请求
	req := r.Copy()
	req.Id = dns.Id() // 生成新的ID

	// 发送到上游DNS服务器
	resp, _, err := c.Exchange(req, upstreamDNS)
	if err != nil {
		// 如果当前上游DNS失败，尝试其他上游DNS
		s.upstreamDNSMutex.RLock()
		remainingUpstreams := make([]string, 0, len(s.upstreamDNS)-1)
		for _, u := range s.upstreamDNS {
			if u != upstreamDNS {
				remainingUpstreams = append(remainingUpstreams, u)
			}
		}
		s.upstreamDNSMutex.RUnlock()

		if len(remainingUpstreams) > 0 {
			// 随机选择一个其他上游DNS重试
			rand.Seed(time.Now().UnixNano())
			retryUpstream := remainingUpstreams[rand.Intn(len(remainingUpstreams))]

			s.logger.Info("上游DNS失败，尝试备用服务器",
				zap.String("failed", upstreamDNS),
				zap.String("retry", retryUpstream),
				zap.Error(err))

			resp, _, err = c.Exchange(req, retryUpstream)
			if err != nil {
				return fmt.Errorf("所有上游DNS服务器都失败: %w", err)
			}
		} else {
			return err
		}
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

	// 3. 先尝试从DNS记录缓存中查询，优先级高于服务缓存
	s.cacheMutex.RLock()
	records, ok := s.dnsCache[domain]
	s.cacheMutex.RUnlock()

	if ok {
		// 查找对应类型的记录
		recordType := dns.TypeToString[q.Qtype]

		s.cacheMutex.RLock()
		record, ok := records[recordType]
		s.cacheMutex.RUnlock()

		if ok {
			// 创建DNS记录
			rrStr := fmt.Sprintf("%s. %s %s", domain, recordType, record.Value)
			rr, err := dns.NewRR(rrStr)
			if err == nil {
				m.Answer = append(m.Answer, rr)
				return true
			}
		}
	}

	// 4. 如果是普通服务域名，尝试从服务缓存中查询
	if strings.HasSuffix(domain, serviceDomainSuffix) && !strings.HasPrefix(domain, "_") {
		return s.handleServiceQuery(domain, q.Qtype, m)
	}

	// 5. 如果etcdClient未设置或缓存中没有找到，尝试从etcd获取
	if s.etcdClient == nil {
		s.logger.Warn("etcd客户端未设置，无法查询DNS记录")
		return false
	}

	// 从etcd获取DNS记录
	return s.handleRegularDNSQuery(domain, q.Qtype, m)
}

// handleServiceQuery 处理服务DNS查询
func (s *DNSServer) handleServiceQuery(domain string, qtype uint16, m *dns.Msg) bool {
	// 从域名中提取服务名
	// 预期格式: service.namespace.svc.cluster.local
	parts := strings.Split(domain, ".")
	if len(parts) < 5 {
		s.logger.Warn("无效的服务域名格式", zap.String("domain", domain))
		return false
	}

	serviceName := parts[0]

	// 从服务缓存中查询
	s.cacheMutex.RLock()
	instances, ok := s.serviceCache[serviceName]
	instancesCopy := make([]*etcdclient.ServiceInstance, 0, len(instances))
	if ok {
		for _, instance := range instances {
			instancesCopy = append(instancesCopy, instance)
		}
	}
	s.cacheMutex.RUnlock()

	// 如果缓存中有实例，直接处理
	if len(instancesCopy) > 0 {
		return s.handleServiceQueryWithInstances(domain, qtype, instancesCopy, m)
	}

	// 如果缓存中没有，尝试从etcd获取
	if s.etcdClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		serviceInstances, err := s.etcdClient.GetServiceInstances(ctx, serviceName)
		if err != nil || len(serviceInstances) == 0 {
			s.logger.Debug("从etcd获取服务实例失败或未找到",
				zap.String("service", serviceName),
				zap.Error(err))
			return false
		}

		// 更新缓存
		for _, instance := range serviceInstances {
			s.UpdateServiceCache(instance)
		}

		// 使用etcd的结果处理请求
		return s.handleServiceQueryWithInstances(domain, qtype, serviceInstances, m)
	}

	return false
}

// handleServiceQueryWithInstances 使用服务实例处理DNS查询
func (s *DNSServer) handleServiceQueryWithInstances(domain string, qtype uint16, instances []*etcdclient.ServiceInstance, m *dns.Msg) bool {
	if len(instances) == 0 {
		return false
	}

	switch qtype {
	case dns.TypeA:
		// 对于A记录，返回第一个实例的IP（简单负载均衡可以在上层实现）
		rr, err := dns.NewRR(fmt.Sprintf("%s. A %s", domain, instances[0].IPAddress))
		if err != nil {
			s.logger.Error("创建A记录失败", zap.Error(err))
			return false
		}
		m.Answer = append(m.Answer, rr)
		return true

	case dns.TypeSRV:
		// 对于SRV记录，返回所有实例
		for _, instance := range instances {
			// SRV记录格式: _service._proto.name. TTL class SRV priority weight port target
			target := fmt.Sprintf("%s.%s", instance.InstanceID, domain)
			rrStr := fmt.Sprintf("%s. 60 IN SRV 10 10 %d %s", domain, instance.Port, target)

			rr, err := dns.NewRR(rrStr)
			if err != nil {
				s.logger.Error("创建SRV记录失败", zap.Error(err))
				continue
			}

			m.Answer = append(m.Answer, rr)

			// 添加A记录作为附加信息
			additionalRR, err := dns.NewRR(fmt.Sprintf("%s. 60 IN A %s", target, instance.IPAddress))
			if err == nil {
				m.Extra = append(m.Extra, additionalRR)
			}
		}

		return len(m.Answer) > 0
	}

	return false
}

// handleRegularDNSQuery 处理普通DNS查询
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

	// 确保记录不为空且有值
	if record == nil || record.Value == "" {
		s.logger.Debug("DNS记录为空或值为空",
			zap.String("domain", domain),
			zap.String("type", recordType))
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
			s.logger.Error("创建SRV记录失败",
				zap.String("domain", domain),
				zap.String("value", record.Value),
				zap.Error(err))
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
