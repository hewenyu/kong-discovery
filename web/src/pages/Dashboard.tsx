import React, { useState, useEffect } from 'react';
import { Box, Card, CardContent, Grid, Typography, Paper, CircularProgress, Alert } from '@mui/material';
import { Storage as StorageIcon, Dns as DnsIcon } from '@mui/icons-material';
import { systemApi } from '../services/api';
import type { SystemStatusResponse } from '../types/system';

const Dashboard: React.FC = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [statusData, setStatusData] = useState<any>(null);

  // 加载系统状态
  const loadSystemStatus = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await systemApi.getSystemStatus() as SystemStatusResponse;
      setStatusData(response.data);
    } catch (err) {
      console.error('加载系统状态失败:', err);
      setError('无法加载系统状态数据，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  // 首次加载
  useEffect(() => {
    loadSystemStatus();
  }, []);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box>
        <Typography variant="h4" sx={{ mb: 3 }}>
          系统概览
        </Typography>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3 }}>
        系统概览
      </Typography>

      <Grid container spacing={3}>
        {/* 统计卡片 */}
        <Grid size={{ xs: 12, sm: 6 ,md: 6}}>
          <Card>
            <CardContent sx={{ display: 'flex', alignItems: 'center' }}>
              <StorageIcon sx={{ fontSize: 40, mr: 2, color: 'primary.main' }} />
              <Box>
                <Typography variant="h4">{statusData.num_services}</Typography>
                <Typography variant="body2" color="text.secondary">注册服务总数</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, sm: 6 ,md: 6}}>
          <Card>
            <CardContent sx={{ display: 'flex', alignItems: 'center' }}>
              <DnsIcon sx={{ fontSize: 40, mr: 2, color: 'success.main' }} />
              <Box>
                <Typography variant="h4">{statusData.num_services}</Typography>
                <Typography variant="body2" color="text.secondary">健康服务数</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* 系统信息 */}
        <Grid size={{ xs: 12, sm: 6 ,md: 6}}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>系统信息</Typography>
              
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="body2" color="text.secondary">运行时间</Typography>
                <Typography variant="body1">{statusData.uptime}</Typography>
              </Box>
              
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="body2" color="text.secondary">服务发现类型</Typography>
                <Typography variant="body1">DNS</Typography>
              </Box>
              
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="body2" color="text.secondary">DNS端口</Typography>
                <Typography variant="body1">53 (UDP/TCP)</Typography>
              </Box>
              
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="body2" color="text.secondary">启动时间</Typography>
                <Typography variant="body1">{new Date(statusData.start_time).toLocaleString()}</Typography>
              </Box>
              
              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography variant="body2" color="text.secondary">系统版本</Typography>
                <Typography variant="body1">{statusData.version}</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* 资源使用情况 */}
        <Grid size={{ xs: 12, sm: 6 ,md: 6}}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>资源使用情况</Typography>
              
              {statusData.resources && Object.entries(statusData.resources).map(([key, value]) => (
                <Box key={key} sx={{ mb: 1 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                    <Typography variant="body2" color="text.secondary">
                      {key.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase())}
                    </Typography>
                    <Typography variant="body2">{value as string}</Typography>
                  </Box>
                  {key.includes('memory') && (
                    <Paper 
                      sx={{ 
                        height: 10, 
                        width: '100%', 
                        bgcolor: 'background.default',
                        overflow: 'hidden',
                        borderRadius: 1
                      }}
                    >
                      <Box 
                        sx={{ 
                          height: '100%', 
                          width: '40%', // 示例值
                          bgcolor: 'primary.main',
                          transition: 'width 1s ease-in-out'
                        }} 
                      />
                    </Paper>
                  )}
                </Box>
              ))}
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default Dashboard; 