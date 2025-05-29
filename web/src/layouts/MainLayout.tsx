import React, { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { 
  Box, 
  Drawer, 
  AppBar, 
  Toolbar, 
  Typography, 
  Divider, 
  List, 
  ListItem, 
  ListItemButton, 
  ListItemIcon, 
  ListItemText,
  IconButton,
  CssBaseline,
  useTheme,
  Paper,
  alpha
} from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import DashboardIcon from '@mui/icons-material/Dashboard';
import ApiIcon from '@mui/icons-material/Api';
import DnsIcon from '@mui/icons-material/Dns';
import SettingsIcon from '@mui/icons-material/Settings';
import BarChartIcon from '@mui/icons-material/BarChart';

const drawerWidth = 240;

const MainLayout: React.FC = () => {
  const theme = useTheme();
  const navigate = useNavigate();
  const location = useLocation();
  const [open, setOpen] = useState(true);

  const handleDrawerOpen = () => {
    setOpen(true);
  };

  const handleDrawerClose = () => {
    setOpen(false);
  };

  const menuItems = [
    { text: '概览', icon: <DashboardIcon />, path: '/' },
    { text: '服务管理', icon: <ApiIcon />, path: '/services' },
    { text: 'DNS管理', icon: <DnsIcon />, path: '/dns' },
    { text: '系统监控', icon: <BarChartIcon />, path: '/metrics' },
    { text: '系统设置', icon: <SettingsIcon />, path: '/settings' },
  ];

  return (
    <Box sx={{ display: 'flex', height: '100vh' }}>
      <CssBaseline />
      <AppBar
        position="fixed"
        elevation={0}
        sx={{
          backgroundColor: '#1155cb',
          zIndex: theme.zIndex.drawer + 1,
          boxShadow: 'none',
          borderBottom: '1px solid #e0e0e0',
          transition: theme.transitions.create(['width', 'margin'], {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen,
          }),
          ...(open && {
            marginLeft: drawerWidth,
            width: `calc(100% - ${drawerWidth}px)`,
            transition: theme.transitions.create(['width', 'margin'], {
              easing: theme.transitions.easing.sharp,
              duration: theme.transitions.duration.enteringScreen,
            }),
          }),
        }}
      >
        <Toolbar>
          <IconButton
            color="inherit"
            aria-label="open drawer"
            onClick={handleDrawerOpen}
            edge="start"
            sx={{
              marginRight: 5,
              ...(open && { display: 'none' }),
            }}
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" noWrap component="div" sx={{ fontWeight: 600 }}>
            Kong 服务发现
            <Box component="span" sx={{ 
              ml: 1, 
              fontSize: '0.7rem', 
              backgroundColor: 'rgba(255,255,255,0.2)', 
              p: '2px 6px', 
              borderRadius: '4px', 
              fontWeight: 'bold' 
            }}>
              DNS
            </Box>
          </Typography>
        </Toolbar>
      </AppBar>
      <Drawer
        variant="permanent"
        open={open}
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          [`& .MuiDrawer-paper`]: {
            backgroundColor: '#0A1929',
            color: 'white',
            width: drawerWidth,
            boxSizing: 'border-box',
            borderRight: 'none',
            ...(open ? {
              transition: theme.transitions.create('width', {
                easing: theme.transitions.easing.sharp,
                duration: theme.transitions.duration.enteringScreen,
              }),
              overflowX: 'hidden',
            } : {
              transition: theme.transitions.create('width', {
                easing: theme.transitions.easing.sharp,
                duration: theme.transitions.duration.leavingScreen,
              }),
              overflowX: 'hidden',
              width: theme.spacing(7),
              [theme.breakpoints.up('sm')]: {
                width: theme.spacing(9),
              },
            }),
          },
        }}
      >
        <Toolbar
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            px: [1],
            backgroundColor: '#071426',
            color: 'white',
            minHeight: '64px !important',
          }}
        >
          {open && (
            <Typography variant="h6" sx={{ fontWeight: 'bold', ml: 2, fontSize: '1.1rem' }}>
              Kong Manager
              <Box component="span" sx={{ 
                ml: 1, 
                fontSize: '0.7rem', 
                backgroundColor: 'rgba(255,255,255,0.2)', 
                p: '2px 6px', 
                borderRadius: '4px',
                verticalAlign: 'middle'
              }}>
                DNS
              </Box>
            </Typography>
          )}
          <IconButton onClick={handleDrawerClose} sx={{ color: 'white' }}>
            <ChevronLeftIcon />
          </IconButton>
        </Toolbar>
        <Box sx={{ mt: 2, mb: 2, mx: 2 }}>
          {/* 空白区域 */}
        </Box>
        <List sx={{ p: 0 }}>
          {menuItems.map((item) => {
            const isSelected = location.pathname === item.path;
            return (
              <ListItem 
                key={item.text} 
                disablePadding 
                sx={{ 
                  display: 'block',
                  mb: 0.5,
                  mx: 1,
                }}
              >
                <ListItemButton
                  sx={{
                    minHeight: 44,
                    justifyContent: open ? 'initial' : 'center',
                    px: 2,
                    py: 1,
                    borderRadius: '4px',
                    backgroundColor: isSelected ? alpha(theme.palette.primary.main, 0.2) : 'transparent',
                    '&:hover': {
                      backgroundColor: isSelected ? alpha(theme.palette.primary.main, 0.3) : alpha(theme.palette.primary.main, 0.1),
                    }
                  }}
                  onClick={() => navigate(item.path)}
                >
                  <ListItemIcon
                    sx={{
                      minWidth: 0,
                      mr: open ? 2 : 'auto',
                      justifyContent: 'center',
                      color: isSelected ? 'white' : 'rgba(255,255,255,0.7)',
                    }}
                  >
                    {item.icon}
                  </ListItemIcon>
                  <ListItemText 
                    primary={item.text} 
                    sx={{ 
                      opacity: open ? 1 : 0,
                      '& .MuiTypography-root': {
                        fontWeight: isSelected ? 500 : 400,
                        color: isSelected ? 'white' : 'rgba(255,255,255,0.7)',
                      }
                    }} 
                  />
                </ListItemButton>
              </ListItem>
            );
          })}
        </List>
      </Drawer>
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 3,
          width: { sm: `calc(100% - ${drawerWidth}px)` },
          mt: 8,
          backgroundColor: '#f7f9fc',
          minHeight: 'calc(100vh - 64px)',
          overflow: 'auto'
        }}
      >
        <Box sx={{ maxWidth: '1400px', mx: 'auto' }}>
          <Outlet />
        </Box>
      </Box>
    </Box>
  );
};

export default MainLayout; 