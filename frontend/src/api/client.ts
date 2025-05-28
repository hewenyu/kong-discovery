import axios from 'axios';
import type { AxiosInstance, AxiosResponse } from 'axios';

// 默认配置 - 使用API前缀区分API请求与前端路由
const API_BASE_URL = '/api'; // 使用/api前缀，由Vite代理转发到后端

// 定义API响应类型
export interface DNSConfigResponse {
  success: boolean;
  configs: {
    upstream_dns: string;
  };
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
  
  // 获取服务详情
  getServiceDetail: (serviceName: string, instanceId: string) => 
    apiClient.get<any, ServiceDetailResponse>(`/admin/services/${serviceName}/${instanceId}`),
};

// DNS配置API接口
export const dnsApi = {
  // 获取DNS配置
  getDNSConfig: () => apiClient.get<any, DNSConfigResponse>('/admin/config/upstream-dns'),
  
  // 更新上游DNS配置
  updateUpstreamDNS: (upstreamDNS: string) => 
    apiClient.put<any, DNSConfigResponse>('/admin/config/upstream-dns', { upstream_dns: upstreamDNS }),
};

export default apiClient; 