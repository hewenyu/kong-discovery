import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  TextField,
  Button,
  Divider,
  Grid,
  Alert,
  CircularProgress,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  List,
  ListItem,
  ListItemText,
  MenuItem
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import SaveIcon from '@mui/icons-material/Save';
import RefreshIcon from '@mui/icons-material/Refresh';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import { dnsApi } from '../services/api';

// DNS配置接口
interface DNSConfig {
  domain_suffix: string;
  ttl: number;
  upstream_dns: string[];
  cache_size: number;
  cache_ttl: number;
  custom_records: DNSCustomRecord[];
}

interface DNSCustomRecord {
  id: string;
  domain: string;
  type: string;
  value: string;
  ttl: number;
}

const DNS: React.FC = () => {
  const [config, setConfig] = useState<DNSConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [newUpstreamDNS, setNewUpstreamDNS] = useState('');
  const [newCustomRecord, setNewCustomRecord] = useState<Omit<DNSCustomRecord, 'id'>>({
    domain: '',
    type: 'A',
    value: '',
    ttl: 60
  });

  // 获取DNS配置
  const fetchDNSConfig = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await dnsApi.getConfig();
      setConfig(response.data.data);
    } catch (err) {
      console.error('获取DNS配置失败:', err);
      setError('获取DNS配置失败，请检查网络连接或服务器状态');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDNSConfig();
  }, []);

  // 保存DNS配置
  const handleSaveConfig = async () => {
    try {
      setLoading(true);
      setError(null);
      setSuccess(null);
      
      if (!config) return;
      
      await dnsApi.updateConfig(config);
      setSuccess('DNS配置保存成功');
      
      // 3秒后清除成功消息
      setTimeout(() => {
        setSuccess(null);
      }, 3000);
    } catch (err) {
      console.error('保存DNS配置失败:', err);
      setError('保存DNS配置失败，请检查输入数据是否正确');
    } finally {
      setLoading(false);
    }
  };

  // 添加上游DNS
  const handleAddUpstreamDNS = () => {
    if (!newUpstreamDNS.trim() || !config) return;
    
    setConfig({
      ...config,
      upstream_dns: [...config.upstream_dns, newUpstreamDNS.trim()]
    });
    
    setNewUpstreamDNS('');
  };

  // 删除上游DNS
  const handleRemoveUpstreamDNS = (index: number) => {
    if (!config) return;
    
    const newUpstreamDNS = [...config.upstream_dns];
    newUpstreamDNS.splice(index, 1);
    
    setConfig({
      ...config,
      upstream_dns: newUpstreamDNS
    });
  };

  // 添加自定义DNS记录
  const handleAddCustomRecord = () => {
    if (!config) return;
    if (!newCustomRecord.domain.trim() || !newCustomRecord.value.trim()) return;
    
    const id = Math.random().toString(36).substring(2, 9);
    
    setConfig({
      ...config,
      custom_records: [
        ...config.custom_records,
        {
          id,
          ...newCustomRecord
        }
      ]
    });
    
    // 重置表单
    setNewCustomRecord({
      domain: '',
      type: 'A',
      value: '',
      ttl: 60
    });
  };

  // 删除自定义DNS记录
  const handleRemoveCustomRecord = (id: string) => {
    if (!config) return;
    
    setConfig({
      ...config,
      custom_records: config.custom_records.filter(record => record.id !== id)
    });
  };

  if (loading && !config) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">DNS管理</Typography>
        <Box>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={fetchDNSConfig}
            sx={{ mr: 2 }}
          >
            刷新
          </Button>
          <Button
            variant="contained"
            startIcon={<SaveIcon />}
            onClick={handleSaveConfig}
            disabled={!config}
          >
            保存配置
          </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {success && (
        <Alert severity="success" sx={{ mb: 2 }}>
          {success}
        </Alert>
      )}

      {config && (
        <Paper sx={{ p: 3 }}>
          <Typography variant="h6" gutterBottom>
            基本配置
          </Typography>
          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: 'repeat(2, 1fr)' }, gap: 3, mb: 4 }}>
            <TextField
              label="服务域名后缀"
              fullWidth
              value={config.domain_suffix}
              onChange={(e) => setConfig({ ...config, domain_suffix: e.target.value })}
              helperText="例如：service.local"
            />
            <TextField
              label="默认TTL（秒）"
              fullWidth
              type="number"
              value={config.ttl}
              onChange={(e) => setConfig({ ...config, ttl: Number(e.target.value) })}
              helperText="记录默认生存时间"
            />
            <TextField
              label="缓存大小"
              fullWidth
              type="number"
              value={config.cache_size}
              onChange={(e) => setConfig({ ...config, cache_size: Number(e.target.value) })}
              helperText="DNS缓存条目数量"
            />
            <TextField
              label="缓存TTL（秒）"
              fullWidth
              type="number"
              value={config.cache_ttl}
              onChange={(e) => setConfig({ ...config, cache_ttl: Number(e.target.value) })}
              helperText="缓存项目生存时间"
            />
          </Box>

          <Divider sx={{ my: 3 }} />

          <Accordion defaultExpanded>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="h6">上游DNS服务器</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Box sx={{ mb: 2 }}>
                <Typography variant="body2" color="textSecondary" gutterBottom>
                  配置上游DNS服务器，用于解析非本地域名
                </Typography>
              </Box>
              
              <Paper sx={{ p: 2, mb: 4 }}>
                <List>
                  {config.upstream_dns.map((dns, index) => (
                    <ListItem
                      key={index}
                      secondaryAction={
                        <IconButton edge="end" onClick={() => handleRemoveUpstreamDNS(index)}>
                          <DeleteIcon />
                        </IconButton>
                      }
                    >
                      <ListItemText primary={dns} />
                    </ListItem>
                  ))}
                  <ListItem>
                    <TextField
                      fullWidth
                      label="添加上游DNS服务器"
                      value={newUpstreamDNS}
                      onChange={(e) => setNewUpstreamDNS(e.target.value)}
                      helperText="例如：8.8.8.8, 114.114.114.114"
                    />
                    <Button
                      sx={{ ml: 1 }}
                      variant="contained"
                      onClick={handleAddUpstreamDNS}
                      disabled={!newUpstreamDNS.trim()}
                    >
                      添加
                    </Button>
                  </ListItem>
                </List>
              </Paper>
            </AccordionDetails>
          </Accordion>

          <Accordion defaultExpanded>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="h6">自定义DNS记录</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Box sx={{ mb: 2 }}>
                <Typography variant="body2" color="textSecondary" gutterBottom>
                  配置静态DNS记录，不受服务发现影响
                </Typography>
              </Box>
              
              <Paper sx={{ p: 2 }}>
                {config.custom_records.map((record, index) => (
                  <Box key={record.id} sx={{ mb: 3 }}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                      <Typography variant="subtitle1">
                        记录 #{index + 1}
                      </Typography>
                      <IconButton size="small" onClick={() => handleRemoveCustomRecord(record.id)}>
                        <DeleteIcon />
                      </IconButton>
                    </Box>
                    <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: '3fr 2fr 4fr 2fr' }, gap: 2 }}>
                      <TextField
                        label="域名"
                        fullWidth
                        value={record.domain}
                        onChange={(e) => setConfig({
                          ...config,
                          custom_records: config.custom_records.map(r =>
                            r.id === record.id ? { ...r, domain: e.target.value } : r
                          )
                        })}
                      />
                      <TextField
                        label="类型"
                        fullWidth
                        select
                        value={record.type}
                        onChange={(e) => setConfig({
                          ...config,
                          custom_records: config.custom_records.map(r =>
                            r.id === record.id ? { ...r, type: e.target.value } : r
                          )
                        })}
                      >
                        {['A', 'CNAME', 'TXT', 'SRV'].map((type) => (
                          <MenuItem key={type} value={type}>
                            {type}
                          </MenuItem>
                        ))}
                      </TextField>
                      <TextField
                        label="值"
                        fullWidth
                        value={record.value}
                        onChange={(e) => setConfig({
                          ...config,
                          custom_records: config.custom_records.map(r =>
                            r.id === record.id ? { ...r, value: e.target.value } : r
                          )
                        })}
                      />
                      <TextField
                        label="TTL（秒）"
                        fullWidth
                        type="number"
                        value={record.ttl}
                        onChange={(e) => setConfig({
                          ...config,
                          custom_records: config.custom_records.map(r =>
                            r.id === record.id ? { ...r, ttl: Number(e.target.value) } : r
                          )
                        })}
                      />
                    </Box>
                  </Box>
                ))}
                <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
                  <Button
                    variant="contained"
                    onClick={handleAddCustomRecord}
                    startIcon={<AddIcon />}
                  >
                    添加记录
                  </Button>
                </Box>
              </Paper>
            </AccordionDetails>
          </Accordion>
        </Paper>
      )}
    </Box>
  );
};

export default DNS; 