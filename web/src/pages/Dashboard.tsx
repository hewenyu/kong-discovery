import React from 'react';
import { Box, Card, CardContent, Grid, Typography, Paper } from '@mui/material';
import { Storage as StorageIcon, Dns as DnsIcon, Timeline as TimelineIcon } from '@mui/icons-material';

const Dashboard: React.FC = () => {
  // 这里仅为示例数据，实际应从API获取
  const dashboardData = {
    totalServices: 24,
    healthyServices: 22,
    dnsQueries: 1245,
    uptime: '5天23小时15分钟',
    serviceTypes: [
      { name: 'API服务', count: 12 },
      { name: '数据库', count: 5 },
      { name: '缓存', count: 3 },
      { name: '其他', count: 4 },
    ]
  };

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3 }}>
        系统概览
      </Typography>

      <Grid container spacing={3}>
        {/* 统计卡片 */}
        <Grid item xs={12} sm={6} md={4}>
          <Card>
            <CardContent sx={{ display: 'flex', alignItems: 'center' }}>
              <StorageIcon sx={{ fontSize: 40, mr: 2, color: 'primary.main' }} />
              <Box>
                <Typography variant="h4">{dashboardData.totalServices}</Typography>
                <Typography variant="body2" color="text.secondary">注册服务总数</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={4}>
          <Card>
            <CardContent sx={{ display: 'flex', alignItems: 'center' }}>
              <DnsIcon sx={{ fontSize: 40, mr: 2, color: 'success.main' }} />
              <Box>
                <Typography variant="h4">{dashboardData.healthyServices}</Typography>
                <Typography variant="body2" color="text.secondary">健康服务数</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={4}>
          <Card>
            <CardContent sx={{ display: 'flex', alignItems: 'center' }}>
              <TimelineIcon sx={{ fontSize: 40, mr: 2, color: 'secondary.main' }} />
              <Box>
                <Typography variant="h4">{dashboardData.dnsQueries}</Typography>
                <Typography variant="body2" color="text.secondary">DNS查询次数(今日)</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* 系统信息 */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>系统信息</Typography>
              
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="body2" color="text.secondary">运行时间</Typography>
                <Typography variant="body1">{dashboardData.uptime}</Typography>
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
                <Typography variant="body2" color="text.secondary">存储引擎</Typography>
                <Typography variant="body1">etcd v3.5</Typography>
              </Box>
              
              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography variant="body2" color="text.secondary">系统版本</Typography>
                <Typography variant="body1">1.0.0</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* 服务类型统计 */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>服务类型统计</Typography>
              
              {dashboardData.serviceTypes.map((type, index) => (
                <Box key={index} sx={{ mb: index !== dashboardData.serviceTypes.length - 1 ? 2 : 0 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                    <Typography variant="body2">{type.name}</Typography>
                    <Typography variant="body2">{type.count}</Typography>
                  </Box>
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
                        width: `${(type.count / dashboardData.totalServices) * 100}%`, 
                        bgcolor: 'primary.main',
                        transition: 'width 1s ease-in-out'
                      }} 
                    />
                  </Paper>
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