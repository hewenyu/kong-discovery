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
  Tooltip,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Stack
} from '@mui/material';
import { 
  Search as SearchIcon, 
  Refresh as RefreshIcon,
  Info as InfoIcon,
  CheckCircle as CheckCircleIcon,
  Cancel as CancelIcon,
  HelpOutline as HelpOutlineIcon,
  Add as AddIcon
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { serviceApi, namespaceApi } from '../services/api';
import type { Service, ServiceListResponse, Namespace, NamespaceListResponse } from '../types/service';
import { HealthStatus } from '../types/service';

const Services: React.FC = () => {
  const [services, setServices] = useState<Service[]>([]);
  const [namespaces, setNamespaces] = useState<Namespace[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [selectedNamespace, setSelectedNamespace] = useState<string>('all');
  const [createDialogOpen, setCreateDialogOpen] = useState<boolean>(false);
  const [newNamespace, setNewNamespace] = useState<{name: string, description: string}>({
    name: '',
    description: ''
  });
  
  const navigate = useNavigate();

  // 加载服务列表
  const loadServices = async () => {
    setLoading(true);
    setError(null);
    try {
      let response;
      if (selectedNamespace === 'all') {
        response = await serviceApi.getServices() as ServiceListResponse;
      } else {
        response = await serviceApi.getServicesByNamespace(selectedNamespace) as ServiceListResponse;
      }
      setServices(response.data.services);
    } catch (err) {
      console.error('加载服务列表失败:', err);
      setError('无法加载服务列表，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  // 加载命名空间列表
  const loadNamespaces = async () => {
    try {
      const response = await namespaceApi.getNamespaces() as NamespaceListResponse;
      setNamespaces(response.data.namespaces);
    } catch (err) {
      console.error('加载命名空间列表失败:', err);
    }
  };

  // 创建新命名空间
  const handleCreateNamespace = async () => {
    try {
      await namespaceApi.createNamespace({
        name: newNamespace.name,
        description: newNamespace.description
      });
      setCreateDialogOpen(false);
      setNewNamespace({ name: '', description: '' });
      loadNamespaces(); // 重新加载命名空间列表
    } catch (err) {
      console.error('创建命名空间失败:', err);
    }
  };

  // 首次加载
  useEffect(() => {
    loadNamespaces();
    loadServices();
  }, []);

  // 当选择的命名空间变化时，重新加载服务列表
  useEffect(() => {
    loadServices();
  }, [selectedNamespace]);

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

      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <FormControl size="small" sx={{ minWidth: 200 }}>
          <InputLabel id="namespace-select-label">命名空间</InputLabel>
          <Select
            labelId="namespace-select-label"
            value={selectedNamespace}
            label="命名空间"
            onChange={(e) => setSelectedNamespace(e.target.value)}
          >
            <MenuItem value="all">所有命名空间</MenuItem>
            {namespaces.map((ns) => (
              <MenuItem key={ns.name} value={ns.name}>
                {ns.name} ({ns.service_count})
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <Button 
          variant="outlined" 
          startIcon={<AddIcon />}
          onClick={() => setCreateDialogOpen(true)}
        >
          新建命名空间
        </Button>
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
                <TableCell>命名空间</TableCell>
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
                    <TableCell>
                      <Chip label={service.namespace || 'default'} size="small" color="primary" />
                    </TableCell>
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
                  <TableCell colSpan={9} align="center">
                    {searchTerm ? '没有找到匹配的服务' : '暂无服务数据'}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* 创建命名空间对话框 */}
      <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)}>
        <DialogTitle>创建新命名空间</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1, minWidth: 400 }}>
            <TextField
              label="命名空间名称"
              fullWidth
              value={newNamespace.name}
              onChange={(e) => setNewNamespace({...newNamespace, name: e.target.value})}
              required
            />
            <TextField
              label="描述"
              fullWidth
              multiline
              rows={2}
              value={newNamespace.description}
              onChange={(e) => setNewNamespace({...newNamespace, description: e.target.value})}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateDialogOpen(false)}>取消</Button>
          <Button 
            onClick={handleCreateNamespace} 
            variant="contained" 
            disabled={!newNamespace.name.trim()}
          >
            创建
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Services; 