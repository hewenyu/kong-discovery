import axios, { AxiosResponse } from 'axios';
import { ServiceListResponse, ServiceDetailResponse, RegisterServiceRequest, ServiceResponse } from '../types/service';
import { SystemStatusResponse, MetricsResponse } from '../types/system';

// API基础配置
const apiClient = axios.create({
  baseURL: process.env.REACT_APP_API_URL || 'http://localhost:9090/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  }
});

// 添加请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    // 这里可以添加认证Token
    const token = localStorage.getItem('api_token');
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 添加响应拦截器
apiClient.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    // 这里可以统一处理错误
    if (error.response && error.response.status === 401) {
      // 未授权处理
      console.error('认证失败，请重新登录');
    }
    return Promise.reject(error);
  }
);

// 服务相关API
export const serviceApi = {
  // 获取服务列表
  getServices: (): Promise<AxiosResponse<ServiceListResponse>> => {
    return apiClient.get('/services');
  },

  // 获取服务详情
  getService: (serviceId: string): Promise<AxiosResponse<ServiceDetailResponse>> => {
    return apiClient.get(`/services/${serviceId}`);
  },

  // 注册服务
  registerService: (service: RegisterServiceRequest): Promise<AxiosResponse<ServiceResponse>> => {
    return apiClient.post('/services', service);
  },

  // 注销服务
  deregisterService: (serviceId: string): Promise<AxiosResponse<ServiceResponse>> => {
    return apiClient.delete(`/services/${serviceId}`);
  },

  // 发送心跳
  sendHeartbeat: (serviceId: string): Promise<AxiosResponse<ServiceResponse>> => {
    return apiClient.put(`/services/${serviceId}/heartbeat`);
  }
};

// 系统相关API
export const systemApi = {
  // 获取系统状态
  getStatus: (): Promise<AxiosResponse<SystemStatusResponse>> => {
    return apiClient.get('/status');
  },

  // 获取健康状态
  getHealth: (): Promise<AxiosResponse<{status: string}>> => {
    return apiClient.get('/health');
  },

  // 获取系统指标
  getMetrics: (): Promise<AxiosResponse<MetricsResponse>> => {
    return apiClient.get('/metrics');
  }
};

// DNS配置相关API
export const dnsApi = {
  // 获取DNS配置
  getConfig: (): Promise<AxiosResponse<any>> => {
    return apiClient.get('/dns/config');
  },

  // 更新DNS配置
  updateConfig: (config: any): Promise<AxiosResponse<any>> => {
    return apiClient.put('/dns/config', config);
  }
}; 