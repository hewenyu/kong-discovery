// 系统资源使用情况
export interface SystemResources {
  cpu_usage: number;
  memory_usage: number;
  disk_usage: number;
}

// 系统状态数据
export interface SystemStatusData {
  status: string;
  version: string;
  start_time: string;
  uptime: string;
  num_services: number;
  resources: SystemResources;
}

// 系统状态响应
export interface SystemStatusResponse {
  code: number;
  message: string;
  data: SystemStatusData;
}

// 健康检查响应
export interface HealthCheckResponse {
  code: number;
  message: string;
  data: {
    status: string;
    components: {
      storage: string;
      dns: string;
    };
  };
}

// 系统指标数据
export interface MetricsData {
  dns_queries: {
    total: number;
    success: number;
    failure: number;
    cache_hit: number;
    cache_miss: number;
  };
  api_requests: {
    register: number;
    deregister: number;
    heartbeat: number;
    query: number;
  };
  services: {
    total: number;
    healthy: number;
    unhealthy: number;
  };
  resources: SystemResources;
}

// 系统指标响应
export interface MetricsResponse {
  code: number;
  message: string;
  data: MetricsData;
}

// DNS配置数据
export interface DnsConfigData {
  domain_suffix: string;
  ttl: number;
  upstream_dns: string[];
  cache_size: number;
  cache_ttl: number;
}

// DNS配置响应
export interface DnsConfigResponse {
  code: number;
  message: string;
  data: DnsConfigData;
} 