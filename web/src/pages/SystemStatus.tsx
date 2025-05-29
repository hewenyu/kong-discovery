import React from 'react';
import { Box, Typography, Paper, Card, CardContent } from '@mui/material';

const SystemStatus: React.FC = () => {
  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3 }}>
        系统状态
      </Typography>
      <Paper sx={{ p: 3 }}>
        <Typography variant="body1">
          系统状态页面开发中...
        </Typography>
      </Paper>
    </Box>
  );
};

export default SystemStatus; 