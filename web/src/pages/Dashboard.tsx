import React, { useEffect, useState } from 'react';
import { Paper, Typography, Box, Card, CardContent, CircularProgress, Alert, Chip } from '@mui/material';
import Grid from '@mui/material';
import { SystemStatus } from '../types/system';
import { Service } from '../types/service';
import { systemApi, serviceApi } from '../services/api';
import DnsIcon from '@mui/icons-material/Dns';
import StorageIcon from '@mui/icons-material/Storage';
import MemoryIcon from '@mui/icons-material/Memory';
import AccessTimeIcon from '@mui/icons-material/AccessTime';

const Dashboard: React.FC = () => {
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [services, setServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        // 获取系统状态
        const statusResponse = await systemApi.getStatus();
        setSystemStatus(statusResponse.data.data);

        // 获取服务列表
        const servicesResponse = await serviceApi.getServices();
        setServices(servicesResponse.data.data.services);
        
        setError(null);
      } catch (err) {
        console.error('获取数据失败:', err);
        setError('获取系统数据失败，请稍后重试');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    // 设置5秒自动刷新
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, []);

  if (loading && !systemStatus) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mt: 2 }}>
        {error}
      </Alert>
    );
  }

  if (!systemStatus) {
    return (
      <Alert severity="warning" sx={{ mt: 2 }}>
        无法获取系统状态
      </Alert>
    );
  }

  // 确保systemStatus.resources存在
  const resources = systemStatus.resources || { cpu_usage: 0, memory_usage: 0, memory_total: 0 };

  // 计算健康服务数量
  const healthyServices = services.filter(s => s.health === 'healthy').length;
  const unhealthyServices = services.length - healthyServices;

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" sx={{ mb: 3 }}>系统概览</Typography>

      {systemStatus && (
        <>
          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(4, 1fr)' }, gap: 3, mb: 4 }}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                  <DnsIcon color="primary" />
                  <Typography variant="h6" sx={{ ml: 1 }}>
                    服务总数
                  </Typography>
                </Box>
                <Typography variant="h4" sx={{ fontWeight: 'bold' }}>
                  {systemStatus.num_services}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  已注册服务实例
                </Typography>
              </CardContent>
            </Card>
            
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                  <StorageIcon color="primary" />
                  <Typography variant="h6" sx={{ ml: 1 }}>
                    运行状态
                  </Typography>
                </Box>
                <Typography variant="h4" sx={{ fontWeight: 'bold' }}>
                  {systemStatus.status}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  系统正常运行
                </Typography>
              </CardContent>
            </Card>
            
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                  <MemoryIcon color="primary" />
                  <Typography variant="h6" sx={{ ml: 1 }}>
                    CPU使用率
                  </Typography>
                </Box>
                <Typography variant="h4" sx={{ fontWeight: 'bold' }}>
                  {resources.cpu_usage?.toFixed(1) || '0.0'}%
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  当前CPU使用率
                </Typography>
              </CardContent>
            </Card>
            
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                  <AccessTimeIcon color="primary" />
                  <Typography variant="h6" sx={{ ml: 1 }}>
                    运行时间
                  </Typography>
                </Box>
                <Typography variant="h4" sx={{ fontWeight: 'bold' }}>
                  {systemStatus.uptime?.split('.')[0] || 'N/A'}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  系统已持续运行
                </Typography>
              </CardContent>
            </Card>
          </Box>

          <Typography variant="h5" sx={{ mb: 2 }}>最近注册的服务</Typography>

          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(4, 1fr)' }, gap: 2 }}>
            {services.slice(0, 4).map((service) => (
              <Paper sx={{ p: 2 }} key={service.id}>
                <Typography variant="subtitle1" gutterBottom>
                  {service.name}
                </Typography>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                  <Typography variant="body2">
                    {service.ip}:{service.port}
                  </Typography>
                  <Chip
                    label={service.health}
                    size="small"
                    color={service.health === 'healthy' ? 'success' : 'error'}
                  />
                </Box>
                <Typography variant="caption" display="block" color="text.secondary">
                  ID: {service.id?.substring(0, 8)}...
                </Typography>
              </Paper>
            ))}
          </Box>
        </>
      )}
    </Box>
  );
};

export default Dashboard; 