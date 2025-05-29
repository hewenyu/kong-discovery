import React, { useEffect, useState } from 'react';
import { Box, Typography, CircularProgress, Alert, Card, CardContent, CardHeader, Divider } from '@mui/material';
import { systemApi } from '../services/api';
import { SystemStatus } from '../types/system';

const MetricsPage: React.FC = () => {
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdateTime, setLastUpdateTime] = useState<string>('');

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        // 获取系统状态
        const statusResponse = await systemApi.getStatus();
        setSystemStatus(statusResponse.data.data);
        setLastUpdateTime(new Date().toLocaleString());
        setError(null);
      } catch (err) {
        console.error('获取系统状态失败:', err);
        setError('获取系统状态失败，请稍后重试');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    // 定时刷新数据
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading && !systemStatus) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
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
        无法获取系统状态数据
      </Alert>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        系统监控指标
      </Typography>
      
      <Box sx={{ mb: 3 }}>
        <Alert severity="info">
          数据每30秒自动刷新一次。最后更新时间: {lastUpdateTime}
        </Alert>
      </Box>
      
      <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: 'repeat(2, 1fr)' }, gap: 3 }}>
        {/* 服务状态卡片 */}
        <Card>
          <CardHeader title="系统状态" />
          <Divider />
          <CardContent>
            <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2 }}>
              <Box>
                <Typography variant="subtitle2">状态</Typography>
                <Typography variant="h4">{systemStatus.status}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">版本</Typography>
                <Typography variant="h4">{systemStatus.version}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">服务数量</Typography>
                <Typography variant="h4">{systemStatus.num_services}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">运行时间</Typography>
                <Typography variant="h4">{systemStatus.uptime?.split('.')[0] || 'N/A'}</Typography>
              </Box>
            </Box>
          </CardContent>
        </Card>

        {/* 资源使用情况卡片 */}
        <Card>
          <CardHeader title="资源使用情况" />
          <Divider />
          <CardContent>
            <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2 }}>
              <Box>
                <Typography variant="subtitle2">内存分配</Typography>
                <Typography variant="h4">{systemStatus.resources?.memory_alloc || '0 MB'}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">堆内存</Typography>
                <Typography variant="h4">{systemStatus.resources?.memory_heap || '0 MB'}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">系统内存</Typography>
                <Typography variant="h4">{systemStatus.resources?.memory_sys || '0 MB'}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">GC次数</Typography>
                <Typography variant="h4">{systemStatus.resources?.num_gc || 0}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">Goroutine数量</Typography>
                <Typography variant="h4">{systemStatus.resources?.num_goroutines || 0}</Typography>
              </Box>
            </Box>
          </CardContent>
        </Card>

        {/* 启动信息卡片 */}
        <Card>
          <CardHeader title="启动信息" />
          <Divider />
          <CardContent>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Box>
                <Typography variant="subtitle2">启动时间</Typography>
                <Typography variant="body1">
                  {new Date(systemStatus.start_time).toLocaleString()}
                </Typography>
              </Box>
            </Box>
          </CardContent>
        </Card>
      </Box>
    </Box>
  );
};

export default MetricsPage; 