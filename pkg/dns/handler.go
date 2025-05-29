package dns

import (
	"context"
	"log"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// Handler DNS请求处理器
type Handler struct {
	recordManager    *RecordManager    // DNS记录管理器
	upstreamResolver *UpstreamResolver // 上游DNS解析器
	cache            *DNSCache         // DNS缓存
	domain           string            // 本地域名后缀
}

// NewHandler 创建DNS请求处理器
func NewHandler(recordManager *RecordManager, upstreamResolver *UpstreamResolver, cache *DNSCache, domain string) *Handler {
	return &Handler{
		recordManager:    recordManager,
		upstreamResolver: upstreamResolver,
		cache:            cache,
		domain:           strings.TrimSuffix(domain, "."),
	}
}

// ServeDNS 处理DNS请求
func (h *Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	// 创建响应消息
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = false

	// 只处理标准查询
	if r.Opcode != dns.OpcodeQuery {
		m.Rcode = dns.RcodeNotImplemented
		w.WriteMsg(m)
		return
	}

	// 获取请求的域名和查询类型
	if len(r.Question) == 0 {
		m.Rcode = dns.RcodeFormatError
		w.WriteMsg(m)
		return
	}

	q := r.Question[0]
	name := strings.ToLower(q.Name)

	// 检查缓存
	cacheKey := GetCacheKey(q)
	if cached := h.cache.Get(cacheKey); cached != nil {
		cached.Id = r.Id
		w.WriteMsg(cached)
		return
	}

	// 处理本地域
	if strings.HasSuffix(name, h.domain+".") || strings.HasSuffix(name, h.domain) {
		h.handleLocalDomain(w, r, m, name, q.Qtype)
		return
	}

	// 转发到上游DNS
	h.handleUpstreamQuery(w, r)
}

// handleLocalDomain 处理本地域名查询
func (h *Handler) handleLocalDomain(w dns.ResponseWriter, r *dns.Msg, m *dns.Msg, name string, qtype uint16) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 从记录管理器获取记录
	records, err := h.recordManager.GetRecords(ctx, name, qtype)
	if err != nil {
		log.Printf("获取DNS记录失败: %v", err)
		m.Rcode = dns.RcodeServerFailure
		w.WriteMsg(m)
		return
	}

	// 如果没有找到记录，设置NXDOMAIN
	if len(records) == 0 {
		m.Rcode = dns.RcodeNameError
		w.WriteMsg(m)
		return
	}

	// 添加记录到响应
	m.Answer = append(m.Answer, records...)
	m.Authoritative = true

	// 缓存响应
	h.cache.Set(GetCacheKey(r.Question[0]), m.Copy())

	// 发送响应
	w.WriteMsg(m)
}

// handleUpstreamQuery 处理上游DNS查询
func (h *Handler) handleUpstreamQuery(w dns.ResponseWriter, r *dns.Msg) {
	// 使用上游解析器
	resp, err := h.upstreamResolver.Resolve(r)
	if err != nil {
		log.Printf("上游DNS查询失败: %v", err)
		m := new(dns.Msg)
		m.SetReply(r)
		m.Rcode = dns.RcodeServerFailure
		w.WriteMsg(m)
		return
	}

	// 发送响应
	w.WriteMsg(resp)
}

// handleAQuery 处理A记录查询
func (h *Handler) handleAQuery(ctx context.Context, name string) (net.IP, error) {
	records, err := h.recordManager.GetRecords(ctx, name, dns.TypeA)
	if err != nil || len(records) == 0 {
		return nil, err
	}

	// 使用简单轮询实现负载均衡
	// 在实际场景中可以实现更复杂的负载均衡策略
	record := records[0]
	if a, ok := record.(*dns.A); ok {
		return a.A, nil
	}

	return nil, nil
}

// handleSRVQuery 处理SRV记录查询
func (h *Handler) handleSRVQuery(ctx context.Context, name string) (*dns.SRV, error) {
	records, err := h.recordManager.GetRecords(ctx, name, dns.TypeSRV)
	if err != nil || len(records) == 0 {
		return nil, err
	}

	// 使用简单轮询实现负载均衡
	record := records[0]
	if srv, ok := record.(*dns.SRV); ok {
		return srv, nil
	}

	return nil, nil
}
