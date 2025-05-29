import React from 'react';
import { Box, Typography, Paper, Card, CardContent } from '@mui/material';

const DnsConfig: React.FC = () => {
  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 3 }}>
        DNS配置
      </Typography>
      <Paper sx={{ p: 3 }}>
        <Typography variant="body1">
          DNS配置页面开发中...
        </Typography>
      </Paper>
    </Box>
  );
};

export default DnsConfig; 