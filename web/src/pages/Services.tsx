import React, { useEffect, useState } from 'react';
import { 
  Box, 
  Typography, 
  Paper, 
  Button, 
  Chip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Grid,
  Alert,
  CircularProgress
} from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import RefreshIcon from '@mui/icons-material/Refresh';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { Service, RegisterServiceRequest } from '../types/service';
import { serviceApi } from '../services/api';

const Services: React.FC = () => {
  const [services, setServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [openRegisterDialog, setOpenRegisterDialog] = useState(false);
  const [openDetailDialog, setOpenDetailDialog] = useState(false);
  const [selectedService, setSelectedService] = useState<Service | null>(null);
  const [newService, setNewService] = useState<{
    name: string;
    ip: string;
    port: number;
    ttl: string;
    tags: string[];
    metadata: Record<string, string>;
  }>({
    name: '',
    ip: '',
    port: 8080,
    ttl: '30s',
    tags: [],
    metadata: {}
  });
  const [tagsInput, setTagsInput] = useState('');
  const [metadataKeyInput, setMetadataKeyInput] = useState('');
  const [metadataValueInput, setMetadataValueInput] = useState('');

  // 获取服务列表
  const fetchServices = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await serviceApi.getServices();
      setServices(response.data.data.services);
    } catch (err) {
      console.error('获取服务列表失败:', err);
      setError('获取服务列表失败，请检查网络连接或服务器状态');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServices();
    // 定时刷新
    const interval = setInterval(fetchServices, 30000);
    return () => clearInterval(interval);
  }, []);

  // 注册新服务
  const handleRegisterService = async () => {
    try {
      setLoading(true);
      await serviceApi.registerService(newService);
      setOpenRegisterDialog(false);
      // 重置表单
      setNewService({
        name: '',
        ip: '',
        port: 8080,
        ttl: '30s',
        tags: [],
        metadata: {}
      });
      setTagsInput('');
      // 刷新服务列表
      await fetchServices();
    } catch (err) {
      console.error('注册服务失败:', err);
      setError('注册服务失败，请检查输入数据是否正确');
    } finally {
      setLoading(false);
    }
  };

  // 注销服务
  const handleDeregisterService = async (serviceId: string) => {
    if (!window.confirm('确定要注销此服务吗？此操作不可恢复。')) {
      return;
    }
    
    try {
      setLoading(true);
      await serviceApi.deregisterService(serviceId);
      // 刷新服务列表
      await fetchServices();
    } catch (err) {
      console.error('注销服务失败:', err);
      setError('注销服务失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  // 查看服务详情
  const handleViewService = async (serviceId: string) => {
    try {
      const response = await serviceApi.getService(serviceId);
      setSelectedService(response.data.data);
      setOpenDetailDialog(true);
    } catch (err) {
      console.error('获取服务详情失败:', err);
      setError('获取服务详情失败，请稍后重试');
    }
  };

  // 添加标签
  const handleAddTag = () => {
    if (tagsInput.trim()) {
      setNewService({
        ...newService,
        tags: [...(newService.tags || []), tagsInput.trim()]
      });
      setTagsInput('');
    }
  };

  // 删除标签
  const handleRemoveTag = (tagToRemove: string) => {
    setNewService({
      ...newService,
      tags: (newService.tags || []).filter(tag => tag !== tagToRemove)
    });
  };

  // 添加元数据
  const handleAddMetadata = () => {
    if (metadataKeyInput.trim()) {
      const updatedMetadata = { ...(newService.metadata || {}) };
      updatedMetadata[metadataKeyInput.trim()] = metadataValueInput.trim();
      setNewService({
        ...newService,
        metadata: updatedMetadata
      });
      setMetadataKeyInput('');
      setMetadataValueInput('');
    }
  };

  // 表格列定义
  const columns: GridColDef[] = [
    { field: 'name', headerName: '服务名称', flex: 1 },
    { field: 'ip', headerName: 'IP地址', flex: 1 },
    { field: 'port', headerName: '端口', width: 100 },
    { 
      field: 'health', 
      headerName: '健康状态', 
      width: 120,
      renderCell: (params: GridRenderCellParams<Service>) => (
        <Chip 
          label={params.value} 
          color={params.value === 'healthy' ? 'success' : 'error'} 
          size="small"
        />
      )
    },
    {
      field: 'tags',
      headerName: '标签',
      flex: 1,
      renderCell: (params: GridRenderCellParams<Service>) => (
        <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
          {params.value && params.value.map((tag: string) => (
            <Chip key={tag} label={tag} size="small" variant="outlined" />
          ))}
        </Box>
      )
    },
    {
      field: 'registered_at',
      headerName: '注册时间',
      flex: 1,
      valueFormatter: (params: any) => new Date(params.value).toLocaleString()
    },
    {
      field: 'actions',
      headerName: '操作',
      width: 120,
      renderCell: (params: GridRenderCellParams<Service>) => (
        <Box>
          <IconButton
            size="small"
            onClick={() => handleViewService(params.row.id)}
            title="查看详情"
          >
            <VisibilityIcon fontSize="small" />
          </IconButton>
          <IconButton
            size="small"
            onClick={() => handleDeregisterService(params.row.id)}
            title="注销服务"
            color="error"
          >
            <DeleteIcon fontSize="small" />
          </IconButton>
        </Box>
      )
    }
  ];

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">服务管理</Typography>
        <Box>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={fetchServices}
            sx={{ mr: 2 }}
          >
            刷新
          </Button>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setOpenRegisterDialog(true)}
          >
            注册服务
          </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Paper sx={{ height: 500, width: '100%' }}>
        <DataGrid
          rows={services}
          columns={columns}
          loading={loading}
          pageSizeOptions={[10, 25, 50]}
          initialState={{
            pagination: { paginationModel: { pageSize: 10 } },
          }}
          disableRowSelectionOnClick
        />
      </Paper>

      {/* 注册服务对话框 */}
      <Dialog
        open={openRegisterDialog}
        onClose={() => setOpenRegisterDialog(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>注册新服务</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)' }, gap: 2, mt: 1 }}>
            <TextField
              label="服务名称"
              fullWidth
              value={newService.name}
              onChange={(e) => setNewService({ ...newService, name: e.target.value })}
            />
            
            <TextField
              label="IP地址"
              fullWidth
              value={newService.ip}
              onChange={(e) => setNewService({ ...newService, ip: e.target.value })}
            />
            
            <TextField
              label="端口"
              fullWidth
              type="number"
              value={newService.port}
              onChange={(e) => setNewService({ ...newService, port: Number(e.target.value) })}
            />
            
            <TextField
              label="TTL (例如: 30s)"
              fullWidth
              value={newService.ttl}
              onChange={(e) => setNewService({ ...newService, ttl: e.target.value })}
            />

            <Box sx={{ gridColumn: '1 / -1' }}>
              <Typography variant="subtitle1" gutterBottom>
                标签
              </Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {newService.tags?.map((tag, index) => (
                  <Chip
                    key={index}
                    label={tag}
                    onDelete={() => handleRemoveTag(tag)}
                    size="small"
                  />
                )) || null}
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  <TextField
                    size="small"
                    value={tagsInput}
                    onChange={(e) => setTagsInput(e.target.value)}
                    placeholder="添加标签"
                    sx={{ width: 150 }}
                  />
                  <Button size="small" onClick={handleAddTag} disabled={!tagsInput.trim()}>
                    添加
                  </Button>
                </Box>
              </Box>
            </Box>

            <Box sx={{ gridColumn: '1 / -1' }}>
              <Typography variant="subtitle1" gutterBottom>
                元数据
              </Typography>
              <Paper variant="outlined" sx={{ p: 2 }}>
                {newService.metadata && Object.entries(newService.metadata || {}).map(([key, value], index) => (
                  <Box key={index} sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                    <TextField
                      size="small"
                      label="键"
                      value={key}
                      disabled
                      sx={{ mr: 1, width: '40%' }}
                    />
                    <TextField
                      size="small"
                      label="值"
                      value={value}
                      onChange={(e) => {
                        const updatedMetadata = { ...(newService.metadata || {}) };
                        updatedMetadata[key] = e.target.value;
                        setNewService({
                          ...newService,
                          metadata: updatedMetadata
                        });
                      }}
                      sx={{ width: '50%' }}
                    />
                    <IconButton
                      size="small"
                      onClick={() => {
                        if (newService.metadata) {
                          const { [key]: _, ...rest } = newService.metadata;
                          setNewService({
                            ...newService,
                            metadata: rest
                          });
                        }
                      }}
                    >
                      <DeleteIcon />
                    </IconButton>
                  </Box>
                ))}
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  <TextField
                    size="small"
                    label="键"
                    value={metadataKeyInput}
                    onChange={(e) => setMetadataKeyInput(e.target.value)}
                    sx={{ mr: 1, width: '40%' }}
                  />
                  <TextField
                    size="small"
                    label="值"
                    value={metadataValueInput}
                    onChange={(e) => setMetadataValueInput(e.target.value)}
                    sx={{ width: '50%' }}
                  />
                  <IconButton
                    size="small"
                    onClick={handleAddMetadata}
                    disabled={!metadataKeyInput.trim()}
                  >
                    <AddIcon />
                  </IconButton>
                </Box>
              </Paper>
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenRegisterDialog(false)}>取消</Button>
          <Button 
            onClick={handleRegisterService}
            variant="contained"
            disabled={!newService.name || !newService.ip || !newService.port}
          >
            注册
          </Button>
        </DialogActions>
      </Dialog>

      {/* 服务详情对话框 */}
      <Dialog
        open={openDetailDialog}
        onClose={() => setOpenDetailDialog(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>服务详情</DialogTitle>
        <DialogContent>
          {selectedService ? (
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)' }, gap: 2, mt: 1 }}>
              <Box>
                <Typography variant="subtitle2">服务ID</Typography>
                <Typography variant="body1">{selectedService.id}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">服务名称</Typography>
                <Typography variant="body1">{selectedService.name}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">IP地址</Typography>
                <Typography variant="body1">{selectedService.ip}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">端口</Typography>
                <Typography variant="body1">{selectedService.port}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">健康状态</Typography>
                <Chip 
                  label={selectedService.health} 
                  color={selectedService.health === 'healthy' ? 'success' : 'error'}
                />
              </Box>
              <Box>
                <Typography variant="subtitle2">TTL</Typography>
                <Typography variant="body1">{selectedService.ttl}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">注册时间</Typography>
                <Typography variant="body1">
                  {new Date(selectedService.registered_at).toLocaleString()}
                </Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">最后心跳时间</Typography>
                <Typography variant="body1">
                  {new Date(selectedService.last_heartbeat).toLocaleString()}
                </Typography>
              </Box>
              <Box sx={{ gridColumn: '1 / -1' }}>
                <Typography variant="subtitle2">标签</Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 1 }}>
                  {selectedService.tags && selectedService.tags.length > 0 ? (
                    selectedService.tags.map((tag, index) => (
                      <Chip key={index} label={tag} size="small" />
                    ))
                  ) : (
                    <Typography variant="body2" color="text.secondary">无标签</Typography>
                  )}
                </Box>
              </Box>
              <Box sx={{ gridColumn: '1 / -1' }}>
                <Typography variant="subtitle2">元数据</Typography>
                <Paper variant="outlined" sx={{ p: 1, mt: 1 }}>
                  {selectedService.metadata && Object.keys(selectedService.metadata).length > 0 ? (
                    <Box component="pre" sx={{ margin: 0, fontSize: '0.875rem', overflowX: 'auto' }}>
                      {JSON.stringify(selectedService.metadata, null, 2)}
                    </Box>
                  ) : (
                    <Typography variant="body2" color="text.secondary">无元数据</Typography>
                  )}
                </Paper>
              </Box>
            </Box>
          ) : (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
              <CircularProgress />
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDetailDialog(false)}>关闭</Button>
          {selectedService && (
            <Button 
              onClick={() => {
                handleDeregisterService(selectedService.id);
                setOpenDetailDialog(false);
              }}
              color="error"
            >
              注销服务
            </Button>
          )}
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Services; 