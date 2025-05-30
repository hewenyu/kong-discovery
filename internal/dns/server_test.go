package dns

import (
	"context"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestDNSServer(t *testing.T) {
	// 使用非标准端口以避免需要root权限
	config := DefaultConfig()
	config.DNSAddr = "127.0.0.1:15353"

	// 创建并启动DNS服务器
	server := NewServer(config)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("启动DNS服务器失败: %v", err)
	}

	// 确保服务器有时间启动
	time.Sleep(500 * time.Millisecond)

	// 创建DNS客户端并测试查询
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion("test.service.local.", dns.TypeA)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, "127.0.0.1:15353")
	if err != nil {
		t.Fatalf("DNS查询失败: %v", err)
	}

	// 检查是否收到响应
	if r == nil {
		t.Fatal("未收到DNS响应")
	}

	// 检查响应代码
	if r.Rcode != dns.RcodeSuccess {
		t.Fatalf("DNS响应错误，代码: %v", r.Rcode)
	}

	// 检查是否有回答
	if len(r.Answer) == 0 {
		t.Fatal("DNS响应中没有回答")
	}

	// 检查A记录
	aRecord, ok := r.Answer[0].(*dns.A)
	if !ok {
		t.Fatalf("响应不是A记录: %T", r.Answer[0])
	}

	// 检查IP地址是否为127.0.0.1
	if aRecord.A.String() != "127.0.0.1" {
		t.Fatalf("A记录IP错误，期望:127.0.0.1，实际:%s", aRecord.A.String())
	}

	// 关闭服务器
	if err := server.Stop(); err != nil {
		t.Fatalf("停止DNS服务器失败: %v", err)
	}
}
