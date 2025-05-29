package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hewenyu/kong-discovery/internal/sdk"
	"github.com/miekg/dns"
)

const (
	Port                = 8888
	KongDiscoveryServer = "http://localhost:8081"
	DNSDiscoveryServer  = "127.0.0.1:6553"
	ServiceName         = "example-service"
	ServiceDomain       = "example-service.default.svc.cluster.local" // 使用符合服务器预期的域名格式
)

// 获取本机IP地址
func getIPAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("获取IP地址失败: %v", err)
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1" // 如果找不到，返回本地回环地址
}

// 验证DNS服务器是否可访问
func checkDNSServer(address string) error {
	fmt.Printf("正在检查DNS服务器 %s 是否可访问...\n", address)

	// 尝试建立UDP连接到DNS服务器
	conn, err := net.DialTimeout("udp", address, 2*time.Second)
	if err != nil {
		return fmt.Errorf("无法连接到DNS服务器: %w", err)
	}
	defer conn.Close()

	fmt.Printf("DNS服务器 %s 连接成功\n", address)
	return nil
}

func main() {
	// 创建上下文，用于管理整个程序的生命周期
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号，用于优雅退出
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("接收到退出信号，正在优雅退出...")
		cancel()
	}()

	// 检查DNS服务器连接
	if err := checkDNSServer(DNSDiscoveryServer); err != nil {
		fmt.Printf("警告: %v\n", err)
		fmt.Println("DNS服务发现可能无法正常工作！")
		fmt.Println("请确保Kong Discovery的DNS服务正在运行并监听端口6553")
	}

	// 创建自定义DNS解析器并设置默认解析器，确保查询使用我们的DNS服务器
	customResolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", DNSDiscoveryServer)
		},
	}
	net.DefaultResolver = customResolver
	fmt.Println("已设置自定义DNS解析器，确保域名查询使用Kong Discovery DNS服务器")

	// 先创建DNS记录，确保服务可以被解析
	err := createDNSRecordExample(ctx)
	if err != nil {
		fmt.Printf("创建DNS记录示例警告: %v\n", err)
		fmt.Println("继续执行其他功能...")
	}

	// 演示服务注册
	err = registerServiceExample(ctx)
	if err != nil {
		log.Fatalf("服务注册示例失败: %v", err)
	}

	// 演示服务发现
	err = discoverServiceExample(ctx)
	if err != nil {
		fmt.Printf("服务发现示例警告: %v\n", err)
		fmt.Println("继续运行其他功能...")
	}

	// 等待用户按下Ctrl+C
	fmt.Println("示例运行中，按下Ctrl+C退出...")
	<-ctx.Done()
	fmt.Println("示例程序已退出")
}

// 创建DNS记录示例
func createDNSRecordExample(ctx context.Context) error {
	fmt.Println("=== 创建DNS记录示例 ===")

	// 创建Kong Discovery客户端
	client := sdk.NewClient(KongDiscoveryServer)

	ip := getIPAddress()
	fmt.Printf("本机IP地址: %s\n", ip)

	// 手动创建DNS A记录
	dnsRecord := &sdk.DNSRecord{
		Domain: ServiceDomain,
		Type:   "A",
		Value:  ip,
		TTL:    60,
	}

	fmt.Printf("正在创建DNS记录: %s %s %s\n", dnsRecord.Domain, dnsRecord.Type, dnsRecord.Value)
	response, err := client.CreateDNSRecord(ctx, dnsRecord)
	if err != nil {
		fmt.Printf("创建DNS记录失败: %v\n", err)
		return err
	}

	fmt.Printf("DNS记录创建成功: %+v\n", response)

	// 创建SRV记录 - 修改记录值格式，确保目标域名以点号结尾
	srvDomain := fmt.Sprintf("_%s._tcp.default.svc.cluster.local", ServiceName)
	// 确保目标域名以点号结尾，符合DNS格式规范
	targetDomain := ServiceDomain
	if !strings.HasSuffix(targetDomain, ".") {
		targetDomain = targetDomain + "."
	}
	srvValue := fmt.Sprintf("10 10 %d %s", Port, targetDomain)

	srvRecord := &sdk.DNSRecord{
		Domain: srvDomain,
		Type:   "SRV",
		Value:  srvValue,
		TTL:    60,
	}

	fmt.Printf("正在创建SRV记录: %s %s %s\n", srvRecord.Domain, srvRecord.Type, srvRecord.Value)
	srvResponse, err := client.CreateDNSRecord(ctx, srvRecord)
	if err != nil {
		fmt.Printf("创建SRV记录失败: %v\n", err)
		// 继续执行，不返回错误
	} else {
		fmt.Printf("SRV记录创建成功: %+v\n", srvResponse)
	}

	// 额外创建一个测试SRV记录，直接按DNS格式要求
	testSrvDomain := fmt.Sprintf("_test-srv._tcp.default.svc.cluster.local")
	testSrvValue := fmt.Sprintf("10 10 %d %s.default.svc.cluster.local.", Port, ServiceName)

	testSrvRecord := &sdk.DNSRecord{
		Domain: testSrvDomain,
		Type:   "SRV",
		Value:  testSrvValue,
		TTL:    60,
	}

	fmt.Printf("正在创建测试SRV记录: %s %s %s\n", testSrvRecord.Domain, testSrvRecord.Type, testSrvRecord.Value)
	testSrvResponse, err := client.CreateDNSRecord(ctx, testSrvRecord)
	if err != nil {
		fmt.Printf("创建测试SRV记录失败: %v\n", err)
	} else {
		fmt.Printf("测试SRV记录创建成功: %+v\n", testSrvResponse)
	}

	return nil
}

// 服务注册示例
func registerServiceExample(ctx context.Context) error {
	fmt.Println("\n=== 服务注册示例 ===")

	// 创建Kong Discovery客户端
	client := sdk.NewClient(KongDiscoveryServer)

	ip := getIPAddress()
	fmt.Printf("本机IP地址: %s\n", ip)

	// 准备服务实例信息
	serviceInstance := &sdk.ServiceInstance{
		ServiceName: ServiceName,
		InstanceID:  fmt.Sprintf("instance-%d", time.Now().Unix()),
		IPAddress:   ip,
		Port:        8080,
		TTL:         60, // 60秒租约
		Metadata: map[string]string{
			"version": "1.0.0",
			"region":  "cn-north",
			"domain":  ServiceDomain, // 添加域名信息，帮助服务器将服务与DNS记录关联
		},
	}

	// 注册服务
	fmt.Printf("正在注册服务: %s, 实例ID: %s\n", serviceInstance.ServiceName, serviceInstance.InstanceID)
	response, err := client.Register(ctx, serviceInstance)
	if err != nil {
		fmt.Printf("服务注册失败: %v\n", err)
		return err
	}

	fmt.Printf("服务注册成功: %+v\n", response)

	// 启动心跳循环，在后台保持服务注册状态
	fmt.Println("启动心跳循环...")
	client.StartHeartbeatLoop(ctx, serviceInstance.ServiceName, serviceInstance.InstanceID, 30*time.Second, 60)

	// 设置一个延迟注销服务的协程
	go func() {
		// 在应用退出前注销服务
		<-ctx.Done()
		// 创建一个新的上下文，因为主上下文已经被取消
		deregisterCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		fmt.Printf("正在注销服务: %s, 实例ID: %s\n", serviceInstance.ServiceName, serviceInstance.InstanceID)
		deregisterResp, err := client.Deregister(deregisterCtx, serviceInstance.ServiceName, serviceInstance.InstanceID)
		if err != nil {
			fmt.Printf("服务注销失败: %v\n", err)
			return
		}
		fmt.Printf("服务注销成功: %+v\n", deregisterResp)

		// 删除DNS记录
		delCtx, delCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer delCancel()

		fmt.Printf("正在删除DNS记录: %s\n", ServiceDomain)
		delResp, err := client.DeleteDNSRecord(delCtx, ServiceDomain, "A")
		if err != nil {
			fmt.Printf("删除DNS记录失败: %v\n", err)
		} else {
			fmt.Printf("DNS记录删除成功: %+v\n", delResp)
		}
	}()

	return nil
}

// 服务发现示例
func discoverServiceExample(ctx context.Context) error {
	fmt.Println("\n=== 服务发现示例 ===")

	// 创建DNS服务发现客户端
	discovery := sdk.NewDNSDiscovery(DNSDiscoveryServer, 60*time.Second)

	// 先尝试解析服务域名
	fmt.Printf("尝试解析服务域名: %s\n", ServiceDomain)
	host, err := discovery.ResolveHost(ctx, ServiceDomain)
	if err != nil {
		fmt.Printf("解析服务域名失败: %v\n", err)
		// 使用dig命令检查DNS记录是否存在
		fmt.Println("使用dig命令检查DNS记录...")
		runDig(ServiceDomain, "A")
		return fmt.Errorf("初始服务解析失败，DNS服务发现可能不可用: %w", err)
	}

	fmt.Printf("成功解析服务域名: %s -> %s\n", ServiceDomain, host)

	// 启动一个周期性查询的协程
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Println("\n--- 定期DNS查询 ---")

				// 解析服务域名 (A记录)
				host, err := discovery.ResolveHost(ctx, ServiceDomain)
				if err != nil {
					fmt.Printf("解析服务域名失败: %v\n", err)
				} else {
					fmt.Printf("解析到服务域名: %s -> %s\n", ServiceDomain, host)
				}

				// 查询原始SRV记录
				fmt.Println("\n1. 查询原始SRV记录:")
				srvDomain := fmt.Sprintf("_%s._tcp.default.svc.cluster.local", ServiceName)
				querySRVRecord(DNSDiscoveryServer, srvDomain)

				// 查询测试SRV记录
				fmt.Println("\n2. 查询测试SRV记录:")
				testSrvDomain := "_test-srv._tcp.default.svc.cluster.local"
				querySRVRecord(DNSDiscoveryServer, testSrvDomain)

				// 直接使用net包进行解析测试
				ips, err := net.LookupIP(ServiceDomain)
				if err != nil {
					fmt.Printf("标准解析失败: %v\n", err)
				} else {
					fmt.Printf("标准解析成功: %s -> %v\n", ServiceDomain, ips)
				}
			}
		}
	}()

	return nil
}

// 查询SRV记录并显示详细信息
func querySRVRecord(dnsServer, domain string) {
	// 直接创建DNS查询消息
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeSRV)
	m.RecursionDesired = true

	// 打印实际发送的DNS查询
	fmt.Printf("发送DNS查询: %s IN SRV\n", dns.Fqdn(domain))

	// 执行DNS查询
	c := new(dns.Client)
	c.Timeout = 5 * time.Second
	r, _, err := c.Exchange(m, dnsServer)

	if err != nil {
		fmt.Printf("SRV查询失败: %v\n", err)
		return
	}

	fmt.Printf("DNS响应状态: %s (Code: %d)\n", dns.RcodeToString[r.Rcode], r.Rcode)

	if r.Rcode == dns.RcodeSuccess && len(r.Answer) > 0 {
		fmt.Printf("找到 %d 条SRV记录:\n", len(r.Answer))
		for i, ans := range r.Answer {
			if srv, ok := ans.(*dns.SRV); ok {
				fmt.Printf("  #%d: %s\n", i+1, srv.String())
				fmt.Printf("     优先级=%d 权重=%d 端口=%d 目标=%s\n",
					srv.Priority, srv.Weight, srv.Port, srv.Target)
			} else {
				fmt.Printf("  #%d: 非SRV记录: %s\n", i+1, ans.String())
			}
		}

		// 显示附加记录
		if len(r.Extra) > 0 {
			fmt.Printf("附加记录 (%d):\n", len(r.Extra))
			for i, extra := range r.Extra {
				fmt.Printf("  额外 #%d: %s\n", i+1, extra.String())
			}
		}
	} else if r.Rcode == dns.RcodeSuccess {
		fmt.Printf("查询成功但没有找到记录\n")
	} else {
		fmt.Printf("查询未返回成功结果\n")

		// 显示权威记录
		if len(r.Ns) > 0 {
			fmt.Printf("权威记录 (%d):\n", len(r.Ns))
			for i, ns := range r.Ns {
				fmt.Printf("  权威 #%d: %s\n", i+1, ns.String())
			}
		}
	}
}

// 运行dig命令检查DNS记录
func runDig(domain, recordType string) {
	fmt.Printf("运行: dig @%s %s %s\n", DNSDiscoveryServer, domain, recordType)
	// 实际上我们不执行命令，这只是一个提示
}
