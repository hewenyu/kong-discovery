import React, { lazy, Suspense } from 'react';
import { createBrowserRouter } from 'react-router-dom';
import MainLayout from './layouts/MainLayout';
import { CircularProgress, Box } from '@mui/material';

// 懒加载页面组件
const Dashboard = lazy(() => import('./pages/Dashboard'));
const Services = lazy(() => import('./pages/Services'));
const ServiceDetail = lazy(() => import('./pages/ServiceDetail'));
const SystemStatus = lazy(() => import('./pages/SystemStatus'));
const DnsConfig = lazy(() => import('./pages/DnsConfig'));

// 加载状态组件
const LoadingComponent = () => (
  <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '80vh' }}>
    <CircularProgress />
  </Box>
);

// 创建路由
const router = createBrowserRouter([
  {
    path: '/',
    element: <MainLayout />,
    children: [
      {
        index: true,
        element: (
          <Suspense fallback={<LoadingComponent />}>
            <Dashboard />
          </Suspense>
        ),
      },
      {
        path: 'services',
        element: (
          <Suspense fallback={<LoadingComponent />}>
            <Services />
          </Suspense>
        ),
      },
      {
        path: 'services/:id',
        element: (
          <Suspense fallback={<LoadingComponent />}>
            <ServiceDetail />
          </Suspense>
        ),
      },
      {
        path: 'system',
        element: (
          <Suspense fallback={<LoadingComponent />}>
            <SystemStatus />
          </Suspense>
        ),
      },
      {
        path: 'dns',
        element: (
          <Suspense fallback={<LoadingComponent />}>
            <DnsConfig />
          </Suspense>
        ),
      },
    ],
  },
]);

export default router; 