package sdk

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// DNSDiscovery 服务发现客户端
type DNSDiscovery struct {
	dnsServer   string
	cacheTTL    time.Duration
	cacheLocker sync.RWMutex
	hostCache   map[string]hostCacheEntry
	srvCache    map[string]srvCacheEntry
}

type hostCacheEntry struct {
	addrs      []string
	expiration time.Time
}

type srvCacheEntry struct {
	targets    []*net.SRV
	expiration time.Time
}

// NewDNSDiscovery 创建DNS服务发现客户端
func NewDNSDiscovery(dnsServer string, cacheTTL time.Duration) *DNSDiscovery {
	// 如果没有指定DNS服务器，默认使用本地Kong Discovery服务
	if dnsServer == "" {
		dnsServer = "127.0.0.1:6553"
	}

	// 如果没有指定缓存TTL，默认使用60秒
	if cacheTTL <= 0 {
		cacheTTL = 60 * time.Second
	}

	fmt.Printf("DNS服务发现客户端初始化，使用DNS服务器: %s\n", dnsServer)

	return &DNSDiscovery{
		dnsServer: dnsServer,
		cacheTTL:  cacheTTL,
		hostCache: make(map[string]hostCacheEntry),
		srvCache:  make(map[string]srvCacheEntry),
	}
}

// ResolveHost 解析主机地址
func (d *DNSDiscovery) ResolveHost(ctx context.Context, serviceName string) (string, error) {
	fmt.Printf("解析主机地址: %s 使用DNS服务器: %s\n", serviceName, d.dnsServer)

	// 检查缓存
	if addr := d.getHostFromCache(serviceName); addr != "" {
		fmt.Printf("使用缓存的主机地址: %s -> %s\n", serviceName, addr)
		return addr, nil
	}

	// 构建要查询的域名
	queryName := serviceName
	if !strings.Contains(serviceName, ".") {
		queryName = serviceName + ".service.discovery"
	}

	fmt.Printf("执行DNS查询: %s\n", queryName)

	// 创建一个新的DNS消息
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(queryName), dns.TypeA)
	m.RecursionDesired = true

	// 发送DNS查询
	c := new(dns.Client)
	c.Timeout = 5 * time.Second

	// 防止长时间阻塞
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 创建一个完成通道
	done := make(chan struct{})
	var r *dns.Msg
	var err error
	var ips []string

	go func() {
		// 执行DNS查询
		r, _, err = c.Exchange(m, d.dnsServer)
		close(done)
	}()

	// 等待查询完成或上下文取消
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("解析服务[%s]超时", queryName)
	case <-done:
		// 继续处理结果
	}

	// 处理查询结果
	if err != nil {
		return "", fmt.Errorf("解析服务[%s]失败: %w", queryName, err)
	}

	if r == nil || r.Rcode != dns.RcodeSuccess {
		return "", fmt.Errorf("未找到服务[%s]的地址", queryName)
	}

	// 解析响应中的A记录
	for _, a := range r.Answer {
		if aRecord, ok := a.(*dns.A); ok {
			ips = append(ips, aRecord.A.String())
		}
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("未找到服务[%s]的地址", queryName)
	}

	fmt.Printf("解析结果: %s -> %v\n", queryName, ips)

	// 更新缓存
	d.updateHostCache(serviceName, ips)

	// 随机选择一个IP返回
	return ips[rand.Intn(len(ips))], nil
}

// ResolveSRV 解析SRV记录
func (d *DNSDiscovery) ResolveSRV(ctx context.Context, serviceName string) (*net.SRV, error) {
	fmt.Printf("解析SRV记录: %s 使用DNS服务器: %s\n", serviceName, d.dnsServer)

	// 检查缓存
	if srv := d.getSRVFromCache(serviceName); srv != nil {
		fmt.Printf("使用缓存的SRV记录: %s\n", serviceName)
		return srv, nil
	}

	// 构建要查询的域名
	queryName := serviceName
	if !strings.Contains(serviceName, ".") {
		queryName = fmt.Sprintf("_%s._tcp.service.discovery", serviceName)
	}

	fmt.Printf("执行SRV查询: %s\n", queryName)

	// 创建一个新的DNS消息
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(queryName), dns.TypeSRV)
	m.RecursionDesired = true

	// 发送DNS查询
	c := new(dns.Client)
	c.Timeout = 5 * time.Second

	// 防止长时间阻塞
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 创建一个完成通道
	done := make(chan struct{})
	var r *dns.Msg
	var err error
	var srvRecords []*net.SRV

	go func() {
		// 执行DNS查询
		r, _, err = c.Exchange(m, d.dnsServer)
		close(done)
	}()

	// 等待查询完成或上下文取消
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("解析SRV记录[%s]超时", queryName)
	case <-done:
		// 继续处理结果
	}

	// 处理查询结果
	if err != nil {
		return nil, fmt.Errorf("解析SRV记录[%s]失败: %w", queryName, err)
	}

	if r == nil || r.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("未找到服务[%s]的SRV记录", queryName)
	}

	// 解析响应中的SRV记录
	for _, a := range r.Answer {
		if srvRecord, ok := a.(*dns.SRV); ok {
			srvRecords = append(srvRecords, &net.SRV{
				Target:   srvRecord.Target,
				Port:     srvRecord.Port,
				Priority: srvRecord.Priority,
				Weight:   srvRecord.Weight,
			})
		}
	}

	if len(srvRecords) == 0 {
		return nil, fmt.Errorf("未找到服务[%s]的SRV记录", queryName)
	}

	fmt.Printf("SRV解析结果: %s -> 找到%d条记录\n", queryName, len(srvRecords))

	// 更新缓存
	d.updateSRVCache(serviceName, srvRecords)

	// 按权重选择一个SRV记录返回
	return selectSRVByWeight(srvRecords), nil
}

// ResolveService 解析服务，返回主机:端口格式
func (d *DNSDiscovery) ResolveService(ctx context.Context, serviceName string) (string, error) {
	// 首先尝试SRV解析
	srv, err := d.ResolveSRV(ctx, serviceName)
	if err == nil && srv != nil {
		// 如果SRV记录存在Target和Port，直接使用
		result := fmt.Sprintf("%s:%d", strings.TrimSuffix(srv.Target, "."), srv.Port)
		fmt.Printf("使用SRV记录解析服务: %s -> %s\n", serviceName, result)
		return result, nil
	}

	// 如果SRV解析失败，尝试A记录解析
	host, err := d.ResolveHost(ctx, serviceName)
	if err != nil {
		return "", err
	}

	// 对于A记录，我们没有端口信息，返回IP地址
	fmt.Printf("使用A记录解析服务: %s -> %s\n", serviceName, host)
	return host, nil
}

// 从缓存中获取主机IP
func (d *DNSDiscovery) getHostFromCache(serviceName string) string {
	d.cacheLocker.RLock()
	defer d.cacheLocker.RUnlock()

	if entry, ok := d.hostCache[serviceName]; ok {
		if time.Now().Before(entry.expiration) {
			// 缓存有效，随机返回一个IP
			return entry.addrs[rand.Intn(len(entry.addrs))]
		}
	}
	return ""
}

// 更新主机缓存
func (d *DNSDiscovery) updateHostCache(serviceName string, ips []string) {
	d.cacheLocker.Lock()
	defer d.cacheLocker.Unlock()

	d.hostCache[serviceName] = hostCacheEntry{
		addrs:      ips,
		expiration: time.Now().Add(d.cacheTTL),
	}
}

// 从缓存中获取SRV记录
func (d *DNSDiscovery) getSRVFromCache(serviceName string) *net.SRV {
	d.cacheLocker.RLock()
	defer d.cacheLocker.RUnlock()

	if entry, ok := d.srvCache[serviceName]; ok {
		if time.Now().Before(entry.expiration) {
			// 缓存有效，按权重选择一个SRV记录
			return selectSRVByWeight(entry.targets)
		}
	}
	return nil
}

// 更新SRV缓存
func (d *DNSDiscovery) updateSRVCache(serviceName string, srvs []*net.SRV) {
	d.cacheLocker.Lock()
	defer d.cacheLocker.Unlock()

	d.srvCache[serviceName] = srvCacheEntry{
		targets:    srvs,
		expiration: time.Now().Add(d.cacheTTL),
	}
}

// 按权重选择SRV记录
func selectSRVByWeight(srvs []*net.SRV) *net.SRV {
	if len(srvs) == 0 {
		return nil
	}

	if len(srvs) == 1 {
		return srvs[0]
	}

	// 计算总权重
	totalWeight := 0
	for _, srv := range srvs {
		totalWeight += int(srv.Weight)
	}

	// 如果所有权重为0，随机选择一个
	if totalWeight == 0 {
		return srvs[rand.Intn(len(srvs))]
	}

	// 按权重随机选择
	n := rand.Intn(totalWeight)
	for _, srv := range srvs {
		n -= int(srv.Weight)
		if n < 0 {
			return srv
		}
	}

	// 理论上不会到这里，但为了安全返回第一个
	return srvs[0]
}
