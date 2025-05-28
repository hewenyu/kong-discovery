import axios from 'axios';
import type { AxiosInstance, AxiosResponse } from 'axios';

// 默认配置 - 使用API前缀区分API请求与前端路由
const API_BASE_URL = '/api'; // 使用/api前缀，由Vite代理转发到后端

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
  getServices: () => apiClient.get('/admin/services'),
  
  // 获取服务详情
  getServiceDetail: (serviceName: string, instanceId: string) => 
    apiClient.get(`/admin/services/${serviceName}/${instanceId}`),
};

export default apiClient; 