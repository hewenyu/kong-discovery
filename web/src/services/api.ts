import axios from 'axios';

// 创建两个axios实例，分别对应不同的API端点
const registerApi = axios.create({
  baseURL: 'http://localhost:8080/api/v1', // 服务注册API端口
  timeout: 10000
});

const adminApi = axios.create({
  baseURL: 'http://localhost:9090/api/v1', // 管理API端口
  timeout: 10000
});

// 服务相关接口
export const serviceApi = {
  // 获取服务列表
  getServices: async () => {
    const response = await adminApi.get('/services');
    return response.data;
  },

  // 获取服务详情
  getServiceById: async (id: string) => {
    const response = await adminApi.get(`/services/${id}`);
    return response.data;
  },

  // 注册服务
  registerService: async (serviceData: any) => {
    const response = await registerApi.post('/services', serviceData);
    return response.data;
  },

  // 注销服务
  deregisterService: async (id: string) => {
    const response = await registerApi.delete(`/services/${id}`);
    return response.data;
  },

  // 发送心跳
  sendHeartbeat: async (id: string) => {
    const response = await registerApi.put(`/services/${id}/heartbeat`);
    return response.data;
  }
};

// 系统相关接口
export const systemApi = {
  // 获取系统状态
  getSystemStatus: async () => {
    const response = await adminApi.get('/status');
    return response.data;
  },

  // 健康检查
  healthCheck: async () => {
    const response = await adminApi.get('/health');
    return response.data;
  },

  // 获取系统指标
  getMetrics: async () => {
    const response = await adminApi.get('/metrics');
    return response.data;
  }
};

// DNS配置相关接口
export const dnsApi = {
  // 获取DNS配置
  getDnsConfig: async () => {
    const response = await adminApi.get('/dns/config');
    return response.data;
  },

  // 更新DNS配置
  updateDnsConfig: async (configData: any) => {
    const response = await adminApi.put('/dns/config', configData);
    return response.data;
  }
};

// 请求拦截器示例
adminApi.interceptors.request.use(
  config => {
    // 可以在这里添加认证token
    // config.headers.Authorization = `Bearer ${getToken()}`;
    return config;
  },
  error => {
    return Promise.reject(error);
  }
);

// 响应拦截器示例
adminApi.interceptors.response.use(
  response => {
    return response;
  },
  error => {
    // 统一错误处理
    if (error.response) {
      // 服务器返回错误
      console.error('API错误:', error.response.data);
    } else if (error.request) {
      // 请求发送但没有收到响应
      console.error('网络错误:', error.request);
    } else {
      // 请求设置时出错
      console.error('请求错误:', error.message);
    }
    return Promise.reject(error);
  }
);

// 对registerApi也应用相同的拦截器
registerApi.interceptors.request.use(
  config => {
    return config;
  },
  error => {
    return Promise.reject(error);
  }
);

registerApi.interceptors.response.use(
  response => {
    return response;
  },
  error => {
    // 统一错误处理
    if (error.response) {
      console.error('API错误:', error.response.data);
    } else if (error.request) {
      console.error('网络错误:', error.request);
    } else {
      console.error('请求错误:', error.message);
    }
    return Promise.reject(error);
  }
); 