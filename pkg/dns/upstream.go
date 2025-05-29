package dns

import (
	"errors"
	"math/rand"
	"time"

	"github.com/miekg/dns"
)

// UpstreamResolver 实现上游DNS解析器
type UpstreamResolver struct {
	servers []string      // 上游DNS服务器列表
	client  *dns.Client   // DNS客户端
	cache   *DNSCache     // DNS缓存
	timeout time.Duration // 超时设置
}

// NewUpstreamResolver 创建上游DNS解析器
func NewUpstreamResolver(servers []string, cache *DNSCache) *UpstreamResolver {
	// 确保存在默认值
	if len(servers) == 0 {
		servers = []string{"8.8.8.8:53", "114.114.114.114:53"}
	}

	return &UpstreamResolver{
		servers: servers,
		client: &dns.Client{
			Net:     "udp",
			Timeout: 5 * time.Second,
		},
		cache:   cache,
		timeout: 5 * time.Second,
	}
}

// Resolve 解析DNS请求
func (ur *UpstreamResolver) Resolve(req *dns.Msg) (*dns.Msg, error) {
	if len(req.Question) == 0 {
		return nil, errors.New("无效的DNS请求：没有问题部分")
	}

	// 检查缓存
	cacheKey := GetCacheKey(req.Question[0])
	if cachedResp := ur.cache.Get(cacheKey); cachedResp != nil {
		// 设置ID以匹配请求
		cachedResp.Id = req.Id
		return cachedResp, nil
	}

	// 随机选择一个上游服务器
	server := ur.randomServer()

	// 发送请求到上游服务器
	resp, rtt, err := ur.client.Exchange(req, server)
	if err != nil {
		// 如果失败，尝试下一个服务器
		if len(ur.servers) > 1 {
			backupServer := ur.randomServerExcept(server)
			resp, rtt, err = ur.client.Exchange(req, backupServer)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// 记录查询时间
	_ = rtt

	// 缓存结果
	if resp != nil && resp.Rcode == dns.RcodeSuccess {
		// 计算TTL，使用响应中最小的TTL
		var ttl time.Duration
		if len(resp.Answer) > 0 {
			// 从回答中获取最小TTL
			minTTL := resp.Answer[0].Header().Ttl
			for _, rr := range resp.Answer {
				if rr.Header().Ttl < minTTL {
					minTTL = rr.Header().Ttl
				}
			}
			ttl = time.Duration(minTTL) * time.Second
		} else {
			// 没有回答时使用默认TTL
			ttl = 60 * time.Second
		}

		ur.cache.SetWithTTL(cacheKey, resp, ttl)
	}

	return resp, nil
}

// randomServer 随机选择一个上游服务器
func (ur *UpstreamResolver) randomServer() string {
	return ur.servers[rand.Intn(len(ur.servers))]
}

// randomServerExcept 随机选择一个不是指定服务器的上游服务器
func (ur *UpstreamResolver) randomServerExcept(except string) string {
	if len(ur.servers) == 1 {
		return ur.servers[0]
	}

	for {
		server := ur.randomServer()
		if server != except {
			return server
		}
	}
}
