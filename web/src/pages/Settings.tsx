import React from 'react';
import { Box, Typography, Paper, Alert, Card, CardContent, CardHeader, Switch, Divider, FormControlLabel, TextField, Button } from '@mui/material';
import SaveIcon from '@mui/icons-material/Save';

const Settings: React.FC = () => {
  return (
    <Box>
      <Typography variant="h4" gutterBottom>
        系统设置
      </Typography>
      
      <Alert severity="info" sx={{ mb: 3 }}>
        系统设置功能将在未来版本中实现，目前仅提供界面预览。
      </Alert>
      
      <Box sx={{ mb: 4 }}>
        <Card>
          <CardHeader title="安全设置" />
          <Divider />
          <CardContent>
            <Box sx={{ mb: 3 }}>
              <FormControlLabel
                control={<Switch checked={true} />}
                label="启用API Token认证"
              />
              <Typography variant="body2" color="textSecondary" sx={{ ml: 4 }}>
                启用后，所有API请求需要携带有效的Token才能访问
              </Typography>
            </Box>
            
            <Box sx={{ mb: 3 }}>
              <FormControlLabel
                control={<Switch checked={true} />}
                label="启用HTTPS"
              />
              <Typography variant="body2" color="textSecondary" sx={{ ml: 4 }}>
                启用后，所有API请求将通过HTTPS加密传输
              </Typography>
            </Box>
            
            <Box sx={{ mb: 3 }}>
              <FormControlLabel
                control={<Switch checked={false} />}
                label="启用IP白名单"
              />
              <Typography variant="body2" color="textSecondary" sx={{ ml: 4 }}>
                启用后，只有指定IP可以访问API接口
              </Typography>
            </Box>
          </CardContent>
        </Card>
      </Box>
      
      <Box sx={{ mb: 4 }}>
        <Card>
          <CardHeader title="端口配置" />
          <Divider />
          <CardContent>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2 }}>
              <TextField
                label="服务注册端口"
                type="number"
                defaultValue={8080}
                helperText="服务注册、注销、心跳接口端口"
                sx={{ width: 200 }}
              />
              
              <TextField
                label="管理端口"
                type="number"
                defaultValue={9090}
                helperText="管理界面API端口"
                sx={{ width: 200 }}
              />
              
              <TextField
                label="DNS服务端口"
                type="number"
                defaultValue={53}
                helperText="DNS解析服务端口"
                sx={{ width: 200 }}
              />
              
              <TextField
                label="前端界面端口"
                type="number"
                defaultValue={3000}
                helperText="Web管理界面端口"
                sx={{ width: 200 }}
              />
            </Box>
          </CardContent>
        </Card>
      </Box>
      
      <Box sx={{ mb: 4 }}>
        <Card>
          <CardHeader title="备份与恢复" />
          <Divider />
          <CardContent>
            <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
              <Button variant="contained" color="primary">
                备份配置
              </Button>
              
              <Button variant="outlined">
                恢复配置
              </Button>
            </Box>
            
            <Box>
              <FormControlLabel
                control={<Switch checked={true} />}
                label="启用自动备份"
              />
              <Typography variant="body2" color="textSecondary" sx={{ ml: 4 }}>
                每天自动备份系统配置和服务数据
              </Typography>
            </Box>
          </CardContent>
        </Card>
      </Box>
      
      <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
        <Button
          variant="contained"
          startIcon={<SaveIcon />}
          size="large"
        >
          保存设置
        </Button>
      </Box>
    </Box>
  );
};

export default Settings; 