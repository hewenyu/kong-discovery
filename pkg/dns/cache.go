package dns

import (
	"sync"
	"time"

	"github.com/miekg/dns"
)

// DNSCache 实现DNS记录缓存
type DNSCache struct {
	mu         sync.RWMutex
	cache      map[string]*cacheEntry
	defaultTTL time.Duration
}

// cacheEntry 表示缓存中的一条记录
type cacheEntry struct {
	msg      *dns.Msg
	expireAt time.Time
}

// NewDNSCache 创建新的DNS缓存
func NewDNSCache(defaultTTL int) *DNSCache {
	return &DNSCache{
		cache:      make(map[string]*cacheEntry),
		defaultTTL: time.Duration(defaultTTL) * time.Second,
	}
}

// Get 从缓存获取DNS响应
func (c *DNSCache) Get(key string) *dns.Msg {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.cache[key]
	if !found {
		return nil
	}

	// 检查是否过期
	if time.Now().After(entry.expireAt) {
		// 异步清理过期记录
		go c.deleteExpired(key)
		return nil
	}

	// 返回缓存副本避免并发修改
	return entry.msg.Copy()
}

// Set 设置缓存记录
func (c *DNSCache) Set(key string, msg *dns.Msg) {
	c.SetWithTTL(key, msg, c.defaultTTL)
}

// SetWithTTL 使用指定TTL设置缓存记录
func (c *DNSCache) SetWithTTL(key string, msg *dns.Msg, ttl time.Duration) {
	if msg == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 创建缓存条目
	c.cache[key] = &cacheEntry{
		msg:      msg.Copy(),
		expireAt: time.Now().Add(ttl),
	}
}

// Delete 从缓存删除记录
func (c *DNSCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
}

// deleteExpired 删除过期记录
func (c *DNSCache) deleteExpired(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 再次检查是否过期（可能在获取锁的过程中已被更新）
	entry, found := c.cache[key]
	if found && time.Now().After(entry.expireAt) {
		delete(c.cache, key)
	}
}

// CleanupExpired 清理所有过期缓存
func (c *DNSCache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.cache {
		if now.After(entry.expireAt) {
			delete(c.cache, key)
		}
	}
}

// GetCacheKey 生成缓存键
func GetCacheKey(q dns.Question) string {
	return q.Name + "-" + dns.TypeToString[q.Qtype]
}

// StartCleanupRoutine 启动定期清理过期缓存的协程
func (c *DNSCache) StartCleanupRoutine(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			c.CleanupExpired()
		}
	}()
}
