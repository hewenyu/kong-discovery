import axios from 'axios';
import type { AxiosInstance, AxiosResponse } from 'axios';

// 默认配置 - 使用API前缀区分API请求与前端路由
const API_BASE_URL = '/api'; // 使用/api前缀，由Vite代理转发到后端

// 定义API响应类型
export interface DNSConfigResponse {
  success: boolean;
  configs: {
    upstream_dns: string[];
  };
  message?: string;
  timestamp: string;
}

// 服务-DNS关联关系响应
export interface ServiceDNSAssociationResponse {
  success: boolean;
  service_name: string;
  associations: Record<string, string[]>; // 域名 -> 记录类型列表
  count: number;
  message?: string;
  timestamp: string;
}

// DNS-服务关联关系响应
export interface DNSServiceAssociationResponse {
  success: boolean;
  domain: string;
  record_type: string;
  services: string[]; // 服务名称列表
  count: number;
  message?: string;
  timestamp: string;
}

// 服务DNS设置
export interface ServiceDNSSettings {
  load_balance_policy: string; // "round-robin", "random", "weighted", "first-only"
  a_ttl: number;
  srv_ttl: number;
  custom_domain?: string;
  instance_weights?: Record<string, number>;
}

// 服务DNS设置响应
export interface ServiceDNSSettingsResponse {
  success: boolean;
  service_name: string;
  settings: ServiceDNSSettings;
  message?: string;
  timestamp: string;
}

export interface ServiceListResponse {
  success: boolean;
  services: string[];
  count: number;
  message?: string;
  timestamp: string;
}

export interface ServiceDetailResponse {
  success: boolean;
  service_name: string;
  instance_id: string;
  ip_address: string;
  port: number;
  ttl: number;
  metadata?: Record<string, string>;
  message?: string;
  last_heartbeat?: string;
  timestamp: string;
}

export interface ServiceInstanceResponse {
  service_name: string;
  instance_id: string;
  ip_address: string;
  port: number;
  status: string;
  last_heartbeat: string;
  metadata?: Record<string, string>;
}

export interface ServiceInstancesResponse {
  success: boolean;
  instances: ServiceInstanceResponse[];
  count: number;
  message?: string;
  timestamp: string;
}

// DNS记录类型
export interface DNSRecord {
  type: string;
  value: string;
  ttl: number;
  tags?: string[];
}

// DNS域名列表响应
export interface DNSDomainsResponse {
  success: boolean;
  domains: string[];
  count: number;
  message?: string;
  timestamp: string;
}

// DNS记录列表响应
export interface DNSRecordsResponse {
  success: boolean;
  domain: string;
  records: Record<string, DNSRecord>;
  count: number;
  message?: string;
  timestamp: string;
}

// DNS记录操作响应
export interface DNSRecordResponse {
  success: boolean;
  domain: string;
  type: string;
  message?: string;
  timestamp: string;
}

// 创建axios实例
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 响应拦截器
apiClient.interceptors.response.use(
  (response: AxiosResponse) => response.data,
  (error) => {
    console.error('API请求错误:', error);
    return Promise.reject(error);
  }
);

// API服务接口
export const serviceApi = {
  // 健康检查
  checkHealth: () => apiClient.get('/health'),
  
  // 获取服务列表
  getServices: () => apiClient.get<any, ServiceListResponse>('/admin/services'),
  
  // 获取所有服务实例
  getAllServiceInstances: () => apiClient.get<any, ServiceInstancesResponse>('/admin/services/instances'),
  
  // 获取服务详情
  getServiceDetail: (serviceName: string, instanceId: string) => 
    apiClient.get<any, ServiceDetailResponse>(`/admin/services/${serviceName}/${instanceId}`),
    
  // 获取服务关联的DNS记录
  getServiceDNSAssociations: (serviceName: string) => 
    apiClient.get<any, ServiceDNSAssociationResponse>(`/admin/services/${serviceName}/dns`),
    
  // 关联DNS记录到服务
  associateDNSWithService: (serviceName: string, domain: string, recordType: string) => 
    apiClient.post<any, any>(`/admin/services/${serviceName}/dns`, { 
      domain, record_type: recordType 
    }),
    
  // 解除DNS记录与服务的关联
  disassociateDNSFromService: (serviceName: string, domain: string, recordType: string) => 
    apiClient.delete<any, any>(`/admin/services/${serviceName}/dns/${domain}/${recordType}`),
    
  // 获取服务DNS设置
  getServiceDNSSettings: (serviceName: string) => 
    apiClient.get<any, ServiceDNSSettingsResponse>(`/admin/services/${serviceName}/dns-settings`),
    
  // 更新服务DNS设置
  updateServiceDNSSettings: (serviceName: string, settings: ServiceDNSSettings) => 
    apiClient.put<any, any>(`/admin/services/${serviceName}/dns-settings`, settings),
};

// DNS配置API接口
export const dnsApi = {
  // 获取DNS配置
  getDNSConfig: () => apiClient.get<any, DNSConfigResponse>('/admin/config/upstream-dns'),
  
  // 更新上游DNS配置
  updateUpstreamDNS: (upstreamDNS: string[]) => 
    apiClient.put<any, DNSConfigResponse>('/admin/config/upstream-dns', { upstream_dns: upstreamDNS }),

  // 获取所有DNS域名
  getAllDNSDomains: () => apiClient.get<any, DNSDomainsResponse>('/admin/dns/domains'),

  // 获取指定域名的所有DNS记录
  getDNSRecords: (domain: string) => 
    apiClient.get<any, DNSRecordsResponse>(`/admin/dns/records/${domain}`),

  // 创建DNS记录
  createDNSRecord: (domain: string, type: string, value: string, ttl: number, tags?: string[]) => 
    apiClient.post<any, DNSRecordResponse>('/admin/dns/records', { 
      domain, type, value, ttl, tags 
    }),

  // 删除DNS记录
  deleteDNSRecord: (domain: string, type: string) => 
    apiClient.delete<any, DNSRecordResponse>(`/admin/dns/records/${domain}/${type}`),
    
  // 获取DNS记录关联的服务
  getDNSServiceAssociations: (domain: string, recordType: string) => 
    apiClient.get<any, DNSServiceAssociationResponse>(`/admin/dns/:${domain}/:${recordType}/services`),
};

export default apiClient; 