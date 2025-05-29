export interface Service {
  id: string;
  name: string;
  ip: string;
  port: number;
  tags: string[];
  metadata: Record<string, string>;
  health: HealthStatus;
  registered_at: string;
  last_heartbeat: string;
  ttl: string;
}

export enum HealthStatus {
  Healthy = "healthy",
  Unhealthy = "unhealthy",
  Unknown = "unknown"
}

export interface DNSRecord {
  id: string;
  domain: string;
  type: string; // A, AAAA, CNAME, SRV
  value: string;
  ttl: number;
  priority?: number;
  weight?: number;
}

export interface RegisterServiceRequest {
  name: string;
  ip: string;
  port: number;
  tags?: string[];
  metadata?: Record<string, string>;
  ttl?: string;
}

export interface ServiceResponse {
  code: number;
  message: string;
  data?: {
    service_id: string;
    registered_at: string;
  };
}

export interface ServiceListResponse {
  code: number;
  message: string;
  data: {
    services: Service[];
  };
}

export interface ServiceDetailResponse {
  code: number;
  message: string;
  data: Service;
} 