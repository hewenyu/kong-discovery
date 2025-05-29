package model

// DNSRecordType 定义DNS记录类型
type DNSRecordType string

const (
	// RecordTypeA A记录，将域名指向IPv4地址
	RecordTypeA DNSRecordType = "A"
	// RecordTypeSRV SRV记录，包含服务名称、协议、端口等信息
	RecordTypeSRV DNSRecordType = "SRV"
)

// DNSRecord 定义DNS记录结构
type DNSRecord struct {
	ID       string        `json:"id"`
	Domain   string        `json:"domain"`             // 完整域名
	Type     DNSRecordType `json:"type"`               // 记录类型：A, SRV等
	Value    string        `json:"value"`              // 对于A记录是IP地址
	TTL      uint32        `json:"ttl"`                // 生存时间，单位秒
	Priority uint16        `json:"priority,omitempty"` // 仅用于SRV记录
	Weight   uint16        `json:"weight,omitempty"`   // 仅用于SRV记录
	Port     uint16        `json:"port,omitempty"`     // 仅用于SRV记录
}

// ServiceDNSRecords 为一个服务生成DNS记录列表
func ServiceDNSRecords(service *Service, domain string, ttl uint32) []*DNSRecord {
	records := make([]*DNSRecord, 0, 2)

	// 生成服务名称
	serviceDomain := service.Name + "." + domain

	// 添加A记录
	aRecord := &DNSRecord{
		ID:     service.ID + "-A",
		Domain: serviceDomain,
		Type:   RecordTypeA,
		Value:  service.IP,
		TTL:    ttl,
	}
	records = append(records, aRecord)

	// 添加SRV记录
	srvRecord := &DNSRecord{
		ID:       service.ID + "-SRV",
		Domain:   "_" + service.Name + "._tcp." + domain,
		Type:     RecordTypeSRV,
		Value:    serviceDomain,
		TTL:      ttl,
		Priority: 10, // 默认优先级
		Weight:   10, // 默认权重
		Port:     uint16(service.Port),
	}
	records = append(records, srvRecord)

	return records
}
