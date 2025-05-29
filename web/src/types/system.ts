export interface SystemStatus {
  status: string;
  version: string;
  start_time: string;
  uptime: string;
  num_services: number;
  resources: ResourceUsage;
}

export interface ResourceUsage {
  memory_alloc?: string;
  memory_heap?: string;
  memory_sys?: string;
  num_gc?: number;
  num_goroutines?: number;
  cpu_usage?: number;
  memory_usage?: number;
  memory_total?: number;
  disk_usage?: number;
  disk_total?: number;
}

export interface SystemStatusResponse {
  code: number;
  message: string;
  data: SystemStatus;
}

export interface Metrics {
  service_metrics?: ServiceMetrics;
  dns_metrics?: DNSMetrics;
  api_metrics?: APIMetrics;
  resource_metrics?: ResourceMetrics;
  historical_metrics?: HistoricalMetrics;
}

export interface ServiceMetrics {
  total_services: number;
  healthy_services: number;
  unhealthy_services: number;
  registrations_per_minute: number;
  deregistrations_per_minute?: number;
}

export interface DNSMetrics {
  queries_per_second: number;
  cache_hit_ratio: number;
  average_response_time: number;
  error_rate: number;
  query_types: {
    type: string;
    value: number;
  }[];
}

export interface APIMetrics {
  requests_per_minute: number;
  average_response_time: number;
  error_rate: number;
  endpoint_stats: {
    endpoint: string;
    requests: number;
    avg_response_time: number;
  }[];
}

export interface ResourceMetrics {
  cpu_usage_history: Array<{time: string, value: number}>;
  memory_usage_history: Array<{time: string, value: number}>;
  service_count_history: Array<{time: string, value: number}>;
}

export interface HistoricalMetrics {
  cpu_usage?: {
    time: string;
    value: number;
  }[];
  memory_usage?: {
    time: string;
    value: number;
  }[];
  service_count?: {
    time: string;
    value: number;
  }[];
  services_over_time?: Array<{
    time: string; 
    value: number;
  }>;
}

export interface MetricsResponse {
  code: number;
  message: string;
  data: Metrics;
}

export interface APIResponse<T> {
  code: number;
  message: string;
  data: T;
} 