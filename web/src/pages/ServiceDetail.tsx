import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  Box, 
  Typography, 
  Paper, 
  Grid, 
  Chip, 
  Card, 
  CardContent, 
  Button, 
  CircularProgress, 
  Divider, 
  Alert,
  IconButton,
  Tooltip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle
} from '@mui/material';
import { 
  ArrowBack as ArrowBackIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  CheckCircle as CheckCircleIcon,
  Cancel as CancelIcon,
  HelpOutline as HelpOutlineIcon,
  Label as LabelIcon
} from '@mui/icons-material';
import { serviceApi } from '../services/api';
import type { Service, ServiceDetailResponse } from '../types/service';
import { HealthStatus } from '../types/service';

const ServiceDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [service, setService] = useState<Service | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState<boolean>(false);
  const [deleteLoading, setDeleteLoading] = useState<boolean>(false);

  // 加载服务详情
  const loadServiceDetail = async () => {
    if (!id) return;

    setLoading(true);
    setError(null);
    try {
      const response = await serviceApi.getServiceById(id) as ServiceDetailResponse;
      setService(response.data);
    } catch (err) {
      console.error('加载服务详情失败:', err);
      setError('无法加载服务详情，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  // 首次加载
  useEffect(() => {
    loadServiceDetail();
  }, [id]);

  // 注销服务
  const handleDeleteService = async () => {
    if (!id) return;

    setDeleteLoading(true);
    try {
      await serviceApi.deregisterService(id);
      setDeleteDialogOpen(false);
      navigate('/services');
    } catch (err) {
      console.error('注销服务失败:', err);
      setError('注销服务失败，请稍后重试');
    } finally {
      setDeleteLoading(false);
    }
  };

  // 发送心跳
  const handleSendHeartbeat = async () => {
    if (!id) return;

    setLoading(true);
    try {
      await serviceApi.sendHeartbeat(id);
      await loadServiceDetail(); // 重新加载服务详情
    } catch (err) {
      console.error('发送心跳失败:', err);
      setError('发送心跳失败，请稍后重试');
      setLoading(false);
    }
  };

  // 渲染健康状态
  const renderHealthStatus = (status: HealthStatus) => {
    switch (status) {
      case HealthStatus.HEALTHY:
        return (
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <CheckCircleIcon sx={{ color: 'success.main', mr: 1 }} />
            <Typography>健康</Typography>
          </Box>
        );
      case HealthStatus.UNHEALTHY:
        return (
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <CancelIcon sx={{ color: 'error.main', mr: 1 }} />
            <Typography>不健康</Typography>
          </Box>
        );
      default:
        return (
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <HelpOutlineIcon sx={{ color: 'warning.main', mr: 1 }} />
            <Typography>未知</Typography>
          </Box>
        );
    }
  };

  if (loading && !service) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error && !service) {
    return (
      <Box>
        <Button 
          startIcon={<ArrowBackIcon />} 
          onClick={() => navigate('/services')}
          sx={{ mb: 2 }}
        >
          返回服务列表
        </Button>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  if (!service) {
    return (
      <Box>
        <Button 
          startIcon={<ArrowBackIcon />} 
          onClick={() => navigate('/services')}
          sx={{ mb: 2 }}
        >
          返回服务列表
        </Button>
        <Alert severity="warning">服务不存在或已被删除</Alert>
      </Box>
    );
  }

  return (
    <Box>
      {/* 顶部操作栏 */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          <IconButton onClick={() => navigate('/services')} sx={{ mr: 1 }}>
            <ArrowBackIcon />
          </IconButton>
          <Typography variant="h4">服务详情</Typography>
        </Box>
        <Box>
          <Tooltip title="刷新">
            <IconButton onClick={loadServiceDetail} sx={{ mr: 1 }}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title="注销服务">
            <IconButton color="error" onClick={() => setDeleteDialogOpen(true)}>
              <DeleteIcon />
            </IconButton>
          </Tooltip>
        </Box>
      </Box>

      {/* 服务基本信息 */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12,md: 6}}>
              <Typography variant="h5" gutterBottom>{service.name}</Typography>
              <Typography variant="body2" color="text.secondary" gutterBottom>
                ID: {service.id}
              </Typography>
              
              <Box sx={{ mt: 2, mb: 2 }}>
                <Typography variant="subtitle2" gutterBottom>健康状态</Typography>
                {renderHealthStatus(service.health)}
              </Box>
            </Grid>

            <Grid size={{ xs: 12 , md: 6}}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="body2" color="text.secondary">IP地址</Typography>
                <Typography variant="body1">{service.ip}</Typography>
              </Box>
              
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="body2" color="text.secondary">端口</Typography>
                <Typography variant="body1">{service.port}</Typography>
              </Box>
              
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="body2" color="text.secondary">注册时间</Typography>
                <Typography variant="body1">{new Date(service.registered_at).toLocaleString()}</Typography>
              </Box>
              
              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography variant="body2" color="text.secondary">最后心跳</Typography>
                <Typography variant="body1">{new Date(service.last_heartbeat).toLocaleString()}</Typography>
              </Box>
            </Grid>
          </Grid>

          <Divider sx={{ my: 2 }} />

          {/* 标签信息 */}
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2" gutterBottom sx={{ display: 'flex', alignItems: 'center' }}>
              <LabelIcon sx={{ mr: 0.5, fontSize: 20 }} />
              标签
            </Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
              {service.tags && service.tags.length > 0 ? (
                service.tags.map((tag, index) => (
                  <Chip key={index} label={tag} size="small" />
                ))
              ) : (
                <Typography variant="body2" color="text.secondary">无标签</Typography>
              )}
            </Box>
          </Box>

          {/* 元数据信息 */}
          <Box>
            <Typography variant="subtitle2" gutterBottom>元数据</Typography>
            {service.metadata && Object.keys(service.metadata).length > 0 ? (
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Grid container spacing={2}>
                  {Object.entries(service.metadata).map(([key, value]) => (
                    <Grid size={{ xs: 12, sm: 6, md: 4 }} key={key}>
                      <Box>
                        <Typography variant="caption" color="text.secondary">{key}</Typography>
                        <Typography variant="body2">{value}</Typography>
                      </Box>
                    </Grid>
                  ))}
                </Grid>
              </Paper>
            ) : (
              <Typography variant="body2" color="text.secondary">无元数据</Typography>
            )}
          </Box>
        </CardContent>
      </Card>

      {/* 心跳操作 */}
      <Card>
        <CardContent>
          <Typography variant="h6" gutterBottom>心跳管理</Typography>
          <Typography variant="body2" color="text.secondary" paragraph>
            手动发送心跳信号可以更新服务的活跃状态，防止服务因超时而被自动清理。
            TTL设置: {service.ttl || '默认值'}
          </Typography>
          <Button 
            variant="contained" 
            color="primary" 
            onClick={handleSendHeartbeat}
            disabled={loading}
            startIcon={loading ? <CircularProgress size={20} /> : null}
          >
            发送心跳
          </Button>
        </CardContent>
      </Card>

      {/* 删除确认对话框 */}
      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
      >
        <DialogTitle>确认注销服务</DialogTitle>
        <DialogContent>
          <DialogContentText>
            您确定要注销服务 <strong>{service.name}</strong> 吗？此操作不可逆，注销后该服务将不再可用。
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>取消</Button>
          <Button 
            onClick={handleDeleteService} 
            color="error" 
            disabled={deleteLoading}
            startIcon={deleteLoading ? <CircularProgress size={20} /> : null}
          >
            确认注销
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ServiceDetail; 