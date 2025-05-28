import { createBrowserRouter } from 'react-router-dom';
import MainLayout from './layouts/MainLayout';
import ServiceList from './pages/ServiceList';
import ServiceDetail from './pages/ServiceDetail';
import DnsConfig from './pages/DnsConfig';
import Settings from './pages/Settings';
import NotFound from './pages/NotFound';

// 创建路由器，使用basename选项适应Vite开发服务器的路由
const router = createBrowserRouter([
  {
    path: '/',
    element: <MainLayout />,
    children: [
      {
        index: true,
        element: <ServiceList />,
      },
      {
        path: 'services',
        element: <ServiceList />,
      },
      {
        path: 'services/:serviceName/:instanceId',
        element: <ServiceDetail />,
      },
      {
        path: 'dns',
        element: <DnsConfig />,
      },
      {
        path: 'settings',
        element: <Settings />,
      },
    ],
  },
  {
    path: '*',
    element: <NotFound />,
  },
], {
  basename: '/' // 确保基础路径正确
});

export default router; 