import React, { useState, useEffect } from 'react';
import { 
  Box, 
  Typography, 
  Paper, 
  Table, 
  TableBody, 
  TableCell, 
  TableContainer, 
  TableHead, 
  TableRow,
  Chip,
  IconButton,
  CircularProgress,
  TextField,
  InputAdornment,
  Tooltip
} from '@mui/material';
import { 
  Search as SearchIcon, 
  Refresh as RefreshIcon,
  Info as InfoIcon,
  CheckCircle as CheckCircleIcon,
  Cancel as CancelIcon,
  HelpOutline as HelpOutlineIcon
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { serviceApi } from '../services/api';
import type { Service, ServiceListResponse } from '../types/service';
import { HealthStatus } from '../types/service';

const Services: React.FC = () => {
  const [services, setServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState<string>('');
  const navigate = useNavigate();

  // 加载服务列表
  const loadServices = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await serviceApi.getServices() as ServiceListResponse;
      setServices(response.data.services);
    } catch (err) {
      console.error('加载服务列表失败:', err);
      setError('无法加载服务列表，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  // 首次加载
  useEffect(() => {
    loadServices();
  }, []);

  // 根据搜索条件过滤服务
  const filteredServices = services.filter(service => 
    service.name.toLowerCase().includes(searchTerm.toLowerCase()) || 
    service.ip.includes(searchTerm) ||
    service.id.includes(searchTerm) ||
    (service.tags && service.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase())))
  );

  // 导航到服务详情页
  const handleServiceClick = (serviceId: string) => {
    navigate(`/services/${serviceId}`);
  };

  // 渲染健康状态
  const renderHealthStatus = (status: HealthStatus) => {
    switch (status) {
      case HealthStatus.HEALTHY:
        return <Chip 
          icon={<CheckCircleIcon />} 
          label="健康" 
          color="success" 
          size="small" 
          variant="outlined"
        />;
      case HealthStatus.UNHEALTHY:
        return <Chip 
          icon={<CancelIcon />} 
          label="不健康" 
          color="error" 
          size="small" 
          variant="outlined"
        />;
      default:
        return <Chip 
          icon={<HelpOutlineIcon />} 
          label="未知" 
          color="warning" 
          size="small" 
          variant="outlined"
        />;
    }
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">服务管理</Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <TextField
            placeholder="搜索服务..."
            variant="outlined"
            size="small"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
          />
          <Tooltip title="刷新列表">
            <IconButton onClick={loadServices} color="primary">
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Box>
      </Box>

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
          <CircularProgress />
        </Box>
      ) : error ? (
        <Paper sx={{ p: 3, bgcolor: 'error.dark', color: 'error.contrastText' }}>
          <Typography>{error}</Typography>
        </Paper>
      ) : (
        <TableContainer component={Paper}>
          <Table sx={{ minWidth: 650 }}>
            <TableHead>
              <TableRow>
                <TableCell>服务名称</TableCell>
                <TableCell>IP地址</TableCell>
                <TableCell>端口</TableCell>
                <TableCell>健康状态</TableCell>
                <TableCell>标签</TableCell>
                <TableCell>注册时间</TableCell>
                <TableCell>最后心跳</TableCell>
                <TableCell>操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredServices.length > 0 ? (
                filteredServices.map((service) => (
                  <TableRow key={service.id} hover>
                    <TableCell>{service.name}</TableCell>
                    <TableCell>{service.ip}</TableCell>
                    <TableCell>{service.port}</TableCell>
                    <TableCell>{renderHealthStatus(service.health)}</TableCell>
                    <TableCell>
                      {service.tags && service.tags.map((tag, index) => (
                        <Chip key={index} label={tag} size="small" sx={{ mr: 0.5 }} />
                      ))}
                    </TableCell>
                    <TableCell>{new Date(service.registered_at).toLocaleString()}</TableCell>
                    <TableCell>{new Date(service.last_heartbeat).toLocaleString()}</TableCell>
                    <TableCell>
                      <Tooltip title="查看详情">
                        <IconButton size="small" onClick={() => handleServiceClick(service.id)}>
                          <InfoIcon />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={8} align="center">
                    {searchTerm ? '没有找到匹配的服务' : '暂无服务数据'}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  );
};

export default Services; 