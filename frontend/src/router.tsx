import { createBrowserRouter } from 'react-router-dom';
import MainLayout from './layouts/MainLayout';
import ServiceList from './pages/ServiceList';
import ServiceDetail from './pages/ServiceDetail';
import DnsConfig from './pages/DnsConfig';
import Settings from './pages/Settings';
import NotFound from './pages/NotFound';

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
]);

export default router; 