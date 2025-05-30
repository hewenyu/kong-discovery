import React, { useState, useEffect } from 'react';
import { 
  Box, 
  Card, 
  CardContent, 
  Typography, 
  LinearProgress, 
  CircularProgress, 
  Alert, 
  Divider, 
  Stack,
  Chip,
  Paper,
  IconButton,
  Tooltip,
  useTheme,
  useMediaQuery
} from '@mui/material';
import { 
  Storage as StorageIcon, 
  Dns as DnsIcon, 
  Memory as MemoryIcon, 
  Speed as SpeedIcon,
  Schedule as ScheduleIcon,
  Build as BuildIcon,
  CloudQueue as CloudIcon,
  Router as RouterIcon,
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  Refresh as RefreshIcon,
  QueryStats as QueryStatsIcon,
  BarChart as BarChartIcon,
  Equalizer as EqualizerIcon
} from '@mui/icons-material';
import { systemApi } from '../services/api';
import type { SystemStatusResponse, MetricsResponse } from '../types/system';

const Dashboard: React.FC = () => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [statusData, setStatusData] = useState<any>(null);
  const [metricsData, setMetricsData] = useState<any>(null);
  const [lastRefreshTime, setLastRefreshTime] = useState<Date>(new Date());

  // 加载系统数据
  const loadSystemData = async () => {
    setLoading(true);
    setError(null);
    try {
      const [statusResponse, metricsResponse] = await Promise.all([
        systemApi.getSystemStatus() as Promise<SystemStatusResponse>,
        systemApi.getMetrics() as Promise<MetricsResponse>
      ]);
      
      setStatusData(statusResponse.data);
      setMetricsData(metricsResponse.data);
      setLastRefreshTime(new Date());
    } catch (err) {
      console.error('加载系统数据失败:', err);
      setError('无法加载系统数据，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  // 首次加载
  useEffect(() => {
    loadSystemData();
    
    // 设置每60秒自动刷新一次
    const interval = setInterval(() => {
      loadSystemData();
    }, 60000);
    
    return () => clearInterval(interval);
  }, []);

  // 将百分比字符串转换为数字（例如"45.2%"转为45.2)
  const parsePercentage = (value: string | number | any): number => {
    // 如果value不是字符串，直接返回数字值或0
    if (typeof value !== 'string') {
      return typeof value === 'number' ? value : 0;
    }
    const match = value.match(/(\d+(\.\d+)?)/);
    return match ? parseFloat(match[1]) : 0;
  };

  // 根据百分比返回颜色
  const getColorByPercentage = (value: number): string => {
    if (value < 50) return 'success.main';
    if (value < 80) return 'warning.main';
    return 'error.main';
  };

  // 计算API请求总数
  const calculateTotalApiRequests = (): number => {
    if (!metricsData?.api_requests) return 0;
    
    // 将对象值转换为数字数组，然后求和
    const apiRequests: Record<string, number> = metricsData.api_requests;
    return Object.values(apiRequests).reduce(
      (sum, val) => sum + (typeof val === 'number' ? val : 0), 
      0
    );
  };

  // 获取DNS查询成功率
  const getDnsSuccessRate = (): number => {
    if (!metricsData?.dns_queries?.total || metricsData.dns_queries.total === 0) {
      return 0;
    }
    return Math.round((metricsData.dns_queries.success / metricsData.dns_queries.total) * 100);
  };

  // 获取DNS缓存命中率
  const getDnsCacheHitRate = (): number => {
    if (!metricsData?.dns_queries?.total || metricsData.dns_queries.total === 0) {
      return 0;
    }
    return Math.round((metricsData.dns_queries.cache_hit / metricsData.dns_queries.total) * 100);
  };

  // 手动刷新
  const handleRefresh = () => {
    loadSystemData();
  };

  if (loading && !statusData) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error && !statusData) {
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
    <div style={{ width: '100%', maxWidth: '100%' }}>
      {/* 标题和状态栏 */}
      <Paper 
        elevation={0} 
        sx={{ 
          p: 2, 
          mb: 2, 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center',
          bgcolor: 'background.paper',
          borderRadius: 1,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          <BarChartIcon sx={{ mr: 1.5, color: 'primary.main' }} />
          <Typography variant="h6" fontWeight="medium">系统概览</Typography>
          <Tooltip title="上次刷新时间">
            <Typography variant="caption" color="text.secondary" sx={{ ml: 2 }}>
              {loading ? '刷新中...' : `上次刷新: ${lastRefreshTime.toLocaleTimeString()}`}
            </Typography>
          </Tooltip>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          <Chip 
            icon={<CloudIcon />} 
            label={statusData?.status === 'running' ? '系统运行中' : statusData?.status} 
            color="success" 
            sx={{ fontWeight: 'medium', mr: 1 }}
          />
          <Tooltip title="刷新数据">
            <IconButton size="small" onClick={handleRefresh} color="primary">
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Box>
      </Paper>

      {/* 使用CSS Grid布局替代Material UI Grid */}
      <div style={{ width: '100%' }}>
        {/* 顶部卡片组 - 使用CSS Grid */}
        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: isMobile ? 'repeat(2, 1fr)' : 'repeat(4, 1fr)', 
          gap: '16px',
          marginBottom: '16px'
        }}>
          {/* 服务总数卡片 */}
          <Card sx={{ 
            bgcolor: 'background.paper', 
            borderRadius: 1, 
            height: '100%',
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            <CardContent sx={{ p: 2, '&:last-child': { pb: 2 }, height: '100%', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <StorageIcon sx={{ fontSize: 36, mr: 1.5, color: 'primary.main' }} />
                <Typography variant="h5" fontWeight="bold">{statusData?.num_services || 0}</Typography>
              </Box>
              <Typography variant="body2" color="text.secondary">注册服务总数</Typography>
            </CardContent>
          </Card>

          {/* 健康服务卡片 */}
          <Card sx={{ 
            bgcolor: 'background.paper', 
            borderRadius: 1, 
            height: '100%',
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            <CardContent sx={{ p: 2, '&:last-child': { pb: 2 }, height: '100%', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <DnsIcon sx={{ fontSize: 36, mr: 1.5, color: 'success.main' }} />
                <Typography variant="h5" fontWeight="bold">
                  {metricsData?.services?.healthy || statusData?.num_services || 0}
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary">健康服务数</Typography>
            </CardContent>
          </Card>

          {/* DNS查询卡片 */}
          <Card sx={{ 
            bgcolor: 'background.paper', 
            borderRadius: 1, 
            height: '100%',
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            <CardContent sx={{ p: 2, '&:last-child': { pb: 2 }, height: '100%', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <SpeedIcon sx={{ fontSize: 36, mr: 1.5, color: 'info.main' }} />
                <Typography variant="h5" fontWeight="bold">
                  {metricsData?.dns_queries?.total || 0}
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary">DNS查询总数</Typography>
            </CardContent>
          </Card>

          {/* 运行时间卡片 */}
          <Card sx={{ 
            bgcolor: 'background.paper', 
            borderRadius: 1, 
            height: '100%',
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            <CardContent sx={{ p: 2, '&:last-child': { pb: 2 }, height: '100%', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <ScheduleIcon sx={{ fontSize: 36, mr: 1.5, color: 'warning.main' }} />
                <Typography variant="h5" fontWeight="bold" noWrap>
                  {statusData?.uptime?.split('.')[0] || 'N/A'}
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary">系统运行时间</Typography>
            </CardContent>
          </Card>
        </div>

        {/* 主要内容区 - 使用CSS Grid */}
        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: isMobile ? '1fr' : 'repeat(3, 1fr)', 
          gap: '16px' 
        }}>
          {/* 网络信息卡片 */}
          <Card sx={{ 
            bgcolor: 'background.paper', 
            borderRadius: 1, 
            height: '100%', 
            mb: 2,
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            <CardContent sx={{ p: 2.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5 }}>
                <RouterIcon sx={{ mr: 1.5, color: 'primary.main', fontSize: 24 }} />
                <Typography variant="subtitle1" fontWeight="medium">网络信息</Typography>
              </Box>
              <Divider sx={{ my: 1.5 }} />
              
              <Stack spacing={2}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" color="text.secondary">DNS端口</Typography>
                  <Chip 
                    label="53 (UDP/TCP)" 
                    size="small" 
                    variant="outlined" 
                    color="primary"
                  />
                </Box>
                
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" color="text.secondary">注册端口</Typography>
                  <Chip 
                    label="8080 (TCP)" 
                    size="small" 
                    variant="outlined" 
                    color="primary"
                  />
                </Box>
                
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" color="text.secondary">管理端口</Typography>
                  <Chip 
                    label="9090 (TCP)" 
                    size="small" 
                    variant="outlined" 
                    color="primary"
                  />
                </Box>
                
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" color="text.secondary">服务发现类型</Typography>
                  <Chip 
                    label="DNS" 
                    size="small" 
                    variant="outlined" 
                    color="success"
                    icon={<CheckCircleIcon />}
                  />
                </Box>
              </Stack>
            </CardContent>
          </Card>
          
          {/* 系统信息卡片 */}
          <Card sx={{ 
            bgcolor: 'background.paper', 
            borderRadius: 1, 
            height: '100%', 
            mb: 2,
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            <CardContent sx={{ p: 2.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5 }}>
                <BuildIcon sx={{ mr: 1.5, color: 'primary.main', fontSize: 24 }} />
                <Typography variant="subtitle1" fontWeight="medium">系统信息</Typography>
              </Box>
              <Divider sx={{ my: 1.5 }} />
              
              <Stack spacing={2}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" color="text.secondary">系统版本</Typography>
                  <Chip 
                    label={statusData?.version || 'N/A'} 
                    size="small" 
                    color="primary" 
                    variant="outlined"
                  />
                </Box>
                
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" color="text.secondary">启动时间</Typography>
                  <Typography variant="body2" fontWeight="medium">
                    {statusData?.start_time ? new Date(statusData.start_time).toLocaleString() : 'N/A'}
                  </Typography>
                </Box>
                
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" color="text.secondary">API请求总数</Typography>
                  <Chip 
                    label={calculateTotalApiRequests()} 
                    size="small" 
                    color="secondary" 
                    variant="outlined"
                  />
                </Box>
              </Stack>
            </CardContent>
          </Card>
          
          {/* 资源使用情况卡片 */}
          <Card sx={{ 
            bgcolor: 'background.paper', 
            borderRadius: 1, 
            height: '100%', 
            mb: 2,
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            <CardContent sx={{ p: 2.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5 }}>
                <MemoryIcon sx={{ mr: 1.5, color: 'primary.main', fontSize: 24 }} />
                <Typography variant="subtitle1" fontWeight="medium">资源使用情况</Typography>
              </Box>
              <Divider sx={{ my: 1.5 }} />
              
              <Stack spacing={2}>
                {statusData?.resources && Object.entries(statusData.resources)
                  .filter(([key]) => ['cpu_usage', 'memory_usage', 'disk_usage'].includes(key))
                  .map(([key, value]) => {
                    const percentValue = parsePercentage(value as string);
                    const colorKey = getColorByPercentage(percentValue);
                    
                    return (
                      <Box key={key}>
                        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 0.75 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center' }}>
                            {percentValue >= 80 && <WarningIcon sx={{ mr: 0.5, fontSize: 16, color: 'warning.main' }} />}
                            <Typography variant="body2" color="text.secondary">
                              {key === 'cpu_usage' && 'CPU 使用率'}
                              {key === 'memory_usage' && '内存使用率'}
                              {key === 'disk_usage' && '磁盘使用率'}
                            </Typography>
                          </Box>
                          <Typography variant="body2" fontWeight="medium">{value as string}</Typography>
                        </Box>
                        <LinearProgress 
                          variant="determinate" 
                          value={percentValue} 
                          sx={{ 
                            height: 8, 
                            borderRadius: 4,
                            bgcolor: 'background.default',
                            '& .MuiLinearProgress-bar': {
                              bgcolor: colorKey
                            }
                          }} 
                        />
                      </Box>
                    );
                  })}
              </Stack>
            </CardContent>
          </Card>
        </div>

        {/* 第二行卡片 */}
        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: isMobile ? '1fr' : 'repeat(3, 1fr)', 
          gap: '16px'
        }}>
          {/* DNS性能统计卡片 */}
          <Card sx={{ 
            bgcolor: 'background.paper', 
            borderRadius: 1, 
            height: '100%',
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            <CardContent sx={{ p: 2.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5 }}>
                <QueryStatsIcon sx={{ mr: 1.5, color: 'primary.main', fontSize: 24 }} />
                <Typography variant="subtitle1" fontWeight="medium">DNS性能</Typography>
              </Box>
              <Divider sx={{ my: 1.5 }} />
              
              {metricsData?.dns_queries ? (
                <Stack spacing={2.5}>
                  <Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.75 }}>
                      <Typography variant="body2" color="text.secondary">查询成功率</Typography>
                      <Typography variant="body2" fontWeight="medium" color="success.main">
                        {getDnsSuccessRate()}%
                      </Typography>
                    </Box>
                    <LinearProgress 
                      variant="determinate" 
                      value={getDnsSuccessRate()} 
                      sx={{ 
                        height: 8, 
                        borderRadius: 4,
                        bgcolor: 'background.default',
                        mb: 1.5
                      }} 
                    />
                    
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.75 }}>
                      <Typography variant="body2" color="text.secondary">缓存命中率</Typography>
                      <Typography variant="body2" fontWeight="medium" color="info.main">
                        {getDnsCacheHitRate()}%
                      </Typography>
                    </Box>
                    <LinearProgress 
                      variant="determinate" 
                      value={getDnsCacheHitRate()} 
                      sx={{ 
                        height: 8, 
                        borderRadius: 4,
                        bgcolor: 'background.default',
                        '& .MuiLinearProgress-bar': {
                          bgcolor: 'info.main'
                        }
                      }} 
                    />
                  </Box>
                  
                  <Box sx={{ 
                    p: 1.5,
                    borderRadius: 2,
                    bgcolor: 'background.default',
                    display: 'grid',
                    gridTemplateColumns: 'repeat(2, 1fr)',
                    gap: 1.5
                  }}>
                    <Box sx={{ textAlign: 'center', p: 1 }}>
                      <Typography variant="caption" color="text.secondary">总查询</Typography>
                      <Typography variant="h6" fontWeight="medium">
                        {metricsData.dns_queries.total || 0}
                      </Typography>
                    </Box>
                    <Box sx={{ textAlign: 'center', p: 1 }}>
                      <Typography variant="caption" color="text.secondary">成功</Typography>
                      <Typography variant="h6" fontWeight="medium" color="success.main">
                        {metricsData.dns_queries.success || 0}
                      </Typography>
                    </Box>
                    <Box sx={{ textAlign: 'center', p: 1 }}>
                      <Typography variant="caption" color="text.secondary">失败</Typography>
                      <Typography variant="h6" fontWeight="medium" color="error.main">
                        {metricsData.dns_queries.failure || 0}
                      </Typography>
                    </Box>
                    <Box sx={{ textAlign: 'center', p: 1 }}>
                      <Typography variant="caption" color="text.secondary">缓存命中</Typography>
                      <Typography variant="h6" fontWeight="medium" color="info.main">
                        {metricsData.dns_queries.cache_hit || 0}
                      </Typography>
                    </Box>
                  </Box>
                </Stack>
              ) : (
                <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', py: 2 }}>
                  暂无DNS查询数据
                </Typography>
              )}
            </CardContent>
          </Card>
          
          {/* 内存统计卡片 */}
          <Card sx={{ 
            bgcolor: 'background.paper', 
            borderRadius: 1, 
            height: '100%',
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            <CardContent sx={{ p: 2.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5 }}>
                <MemoryIcon sx={{ mr: 1.5, color: 'primary.main', fontSize: 24 }} />
                <Typography variant="subtitle1" fontWeight="medium">内存统计</Typography>
              </Box>
              <Divider sx={{ my: 1.5 }} />
              
              <Stack spacing={2}>
                {statusData?.resources && Object.entries(statusData.resources)
                  .filter(([key]) => key.toLowerCase().includes('memory'))
                  .map(([key, value]) => (
                    <Box key={key} sx={{ 
                      p: 1.5, 
                      borderRadius: 2, 
                      bgcolor: 'background.default'
                    }}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Typography variant="body2" color="text.secondary">
                          {key.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase())}
                        </Typography>
                        <Box sx={{ display: 'flex', alignItems: 'center' }}>
                          <Typography variant="body2" fontWeight="bold">
                            {typeof value === 'string' ? value : String(value)}
                          </Typography>
                        </Box>
                      </Box>
                    </Box>
                  ))}
              </Stack>
            </CardContent>
          </Card>
          
          {/* Go协程统计卡片 */}
          <Card sx={{ 
            bgcolor: 'background.paper', 
            borderRadius: 1, 
            height: '100%',
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}>
            <CardContent sx={{ p: 2.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5 }}>
                <EqualizerIcon sx={{ mr: 1.5, color: 'primary.main', fontSize: 24 }} />
                <Typography variant="subtitle1" fontWeight="medium">Go协程统计</Typography>
              </Box>
              <Divider sx={{ my: 1.5 }} />
              
              <Stack spacing={2}>
                {statusData?.resources && Object.entries(statusData.resources)
                  .filter(([key]) => key.toLowerCase().includes('num'))
                  .map(([key, value]) => (
                    <Box key={key} sx={{ 
                      p: 1.5, 
                      borderRadius: 2, 
                      bgcolor: 'background.default',
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center'
                    }}>
                      <Typography variant="body2" color="text.secondary">
                        {key.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase())}
                      </Typography>
                      <Chip 
                        label={value as string || '0'} 
                        size="small" 
                        color={key.includes('Goroutines') ? 'secondary' : 'primary'} 
                        variant="outlined"
                        sx={{ fontWeight: 'bold', borderRadius: 4 }}
                      />
                    </Box>
                  ))}
              </Stack>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default Dashboard; 