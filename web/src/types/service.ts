// 服务实体类型
export interface Service {
  id: string;
  name: string;
  ip: string;
  port: number;
  tags?: string[];
  metadata?: Record<string, string>;
  health: HealthStatus;
  registered_at: string;
  last_heartbeat: string;
  ttl?: string;
}

// 健康状态枚举
export enum HealthStatus {
  HEALTHY = 'healthy',
  UNHEALTHY = 'unhealthy',
  UNKNOWN = 'unknown'
}

// 服务注册请求
export interface RegisterServiceRequest {
  name: string;
  ip: string;
  port: number;
  tags?: string[];
  metadata?: Record<string, string>;
  ttl?: string;
}

// 服务列表响应
export interface ServiceListResponse {
  code: number;
  message: string;
  data: {
    services: Service[];
  };
}

// 服务详情响应
export interface ServiceDetailResponse {
  code: number;
  message: string;
  data: Service;
}

// 服务注册响应
export interface RegisterServiceResponse {
  code: number;
  message: string;
  data: {
    service_id: string;
    registered_at: string;
  };
}

// 心跳响应
export interface HeartbeatResponse {
  code: number;
  message: string;
  data: {
    last_heartbeat: string;
  };
} 