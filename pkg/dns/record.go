package dns

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/miekg/dns"

	"github.com/hewenyu/kong-discovery/pkg/storage"
)

// RecordManager 管理DNS记录
type RecordManager struct {
	storage          storage.ServiceStorage
	namespaceStorage storage.NamespaceStorage
	domain           string
	defaultTTL       uint32
	recordCache      sync.Map // 本地记录缓存，key为域名，value为dns.RR
}

// NewRecordManager 创建DNS记录管理器
func NewRecordManager(storage storage.ServiceStorage, namespaceStorage storage.NamespaceStorage, domain string, ttl int) *RecordManager {
	return &RecordManager{
		storage:          storage,
		namespaceStorage: namespaceStorage,
		domain:           domain,
		defaultTTL:       uint32(ttl),
	}
}

// GetRecords 获取指定域名和类型的DNS记录
func (rm *RecordManager) GetRecords(ctx context.Context, name string, qtype uint16) ([]dns.RR, error) {
	// 首先尝试从缓存获取记录
	if rrs, ok := rm.recordCache.Load(cacheKey(name, qtype)); ok {
		return rrs.([]dns.RR), nil
	}

	// 规范化域名以去除末尾的点
	name = strings.TrimSuffix(name, ".")
	baseDomain := strings.TrimSuffix(rm.domain, ".")

	// 检查域名是否属于我们的域
	if !strings.HasSuffix(name, baseDomain) {
		return nil, nil
	}

	// 解析服务名称和命名空间
	serviceName, namespace := extractServiceInfo(name, baseDomain)
	if serviceName == "" {
		return nil, nil
	}

	var services []*storage.Service
	var err error

	// 根据命名空间决定如何查询服务
	if namespace != "" {
		// 查询指定命名空间下的服务
		services, err = rm.storage.ListServicesByNameAndNamespace(ctx, namespace, serviceName)
	} else {
		// 查询所有命名空间下的指定服务
		services, err = rm.storage.ListServicesByName(ctx, serviceName)
	}

	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, nil
	}

	// 生成DNS记录
	var records []dns.RR

	switch qtype {
	case dns.TypeA:
		// 生成A记录
		for _, service := range services {
			rr, err := createARecord(name, service.IP, rm.defaultTTL)
			if err != nil {
				continue
			}
			records = append(records, rr)
		}
	case dns.TypeSRV:
		// 生成SRV记录
		for _, service := range services {
			// 根据是否指定了命名空间生成不同的目标域名
			var target string
			if namespace != "" {
				target = fmt.Sprintf("%s.%s.%s", service.Name, service.Namespace, baseDomain)
			} else {
				target = fmt.Sprintf("%s.%s", service.Name, baseDomain)
			}

			rr, err := createSRVRecord(name, target, service.Port, rm.defaultTTL)
			if err != nil {
				continue
			}
			records = append(records, rr)
		}
	}

	// 缓存结果
	if len(records) > 0 {
		rm.recordCache.Store(cacheKey(name, qtype), records)
	}

	return records, nil
}

// RefreshRecords 刷新DNS记录缓存
func (rm *RecordManager) RefreshRecords(ctx context.Context) error {
	// 清除缓存
	rm.recordCache = sync.Map{}

	// 获取所有服务
	services, err := rm.storage.ListServices(ctx)
	if err != nil {
		return err
	}

	// 为每个服务预生成记录并缓存
	for _, service := range services {
		// 生成普通域名和带命名空间的域名
		domainName := service.Name + "." + rm.domain
		namespacedDomainName := fmt.Sprintf("%s.%s.%s", service.Name, service.Namespace, rm.domain)

		// 生成标准A记录
		aRecord, err := createARecord(domainName, service.IP, rm.defaultTTL)
		if err == nil {
			rm.addToCache(domainName, dns.TypeA, aRecord)
		}

		// 生成带命名空间的A记录
		namespacedARecord, err := createARecord(namespacedDomainName, service.IP, rm.defaultTTL)
		if err == nil {
			rm.addToCache(namespacedDomainName, dns.TypeA, namespacedARecord)
		}

		// 生成标准SRV记录
		srvDomain := fmt.Sprintf("_%s._tcp.%s", service.Name, rm.domain)
		srvRecord, err := createSRVRecord(srvDomain, domainName, service.Port, rm.defaultTTL)
		if err == nil {
			rm.addToCache(srvDomain, dns.TypeSRV, srvRecord)
		}

		// 生成带命名空间的SRV记录
		namespacedSrvDomain := fmt.Sprintf("_%s._tcp.%s.%s", service.Name, service.Namespace, rm.domain)
		namespacedSrvRecord, err := createSRVRecord(namespacedSrvDomain, namespacedDomainName, service.Port, rm.defaultTTL)
		if err == nil {
			rm.addToCache(namespacedSrvDomain, dns.TypeSRV, namespacedSrvRecord)
		}
	}

	return nil
}

// addToCache 添加记录到缓存
func (rm *RecordManager) addToCache(name string, qtype uint16, rr dns.RR) {
	key := cacheKey(name, qtype)

	var records []dns.RR
	if existingRRs, ok := rm.recordCache.Load(key); ok {
		records = existingRRs.([]dns.RR)
	}

	records = append(records, rr)
	rm.recordCache.Store(key, records)
}

// cacheKey 生成缓存键
func cacheKey(name string, qtype uint16) string {
	return name + "-" + dns.TypeToString[qtype]
}

// extractServiceInfo 从域名中提取服务名称和命名空间
func extractServiceInfo(name, baseDomain string) (serviceName, namespace string) {
	// 处理SRV记录
	if strings.HasPrefix(name, "_") {
		parts := strings.Split(name, ".")
		if len(parts) >= 3 && parts[1] == "_tcp" {
			serviceName = strings.TrimPrefix(parts[0], "_")

			// 检查是否包含命名空间
			if len(parts) >= 4 && parts[2] != baseDomain {
				namespace = parts[2]
			}

			return serviceName, namespace
		}
		return "", ""
	}

	// 处理A记录
	prefix := strings.TrimSuffix(name, "."+baseDomain)
	parts := strings.Split(prefix, ".")
	if len(parts) == 0 {
		return "", ""
	}

	// 如果只有一部分，那就是服务名称
	if len(parts) == 1 {
		return parts[0], ""
	}

	// 如果有两部分，第一部分是服务名称，第二部分是命名空间
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	// 如果有更多部分，先处理第一部分为服务名称，最后一部分为命名空间
	return parts[0], parts[len(parts)-1]
}

// createARecord 创建A记录
func createARecord(name, ip string, ttl uint32) (dns.RR, error) {
	return dns.NewRR(fmt.Sprintf("%s %d IN A %s", name, ttl, ip))
}

// createSRVRecord 创建SRV记录
func createSRVRecord(name, target string, port int, ttl uint32) (dns.RR, error) {
	return dns.NewRR(fmt.Sprintf("%s %d IN SRV 10 10 %d %s", name, ttl, port, target))
}
