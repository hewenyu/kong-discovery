import { useState } from 'react';
import { Layout, Menu, theme, Typography } from 'antd';
import { Link, Outlet, useLocation } from 'react-router-dom';
import {
  ApiOutlined,
  AppstoreOutlined,
  SettingOutlined,
  DashboardOutlined,
  GlobalOutlined,
  CloudServerOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import './mainLayout.css';

const { Header, Content, Sider } = Layout;
const { Title } = Typography;

// 定义菜单项类型
type MenuItem = Required<MenuProps>['items'][number];

// 创建菜单项函数
const getItem = (
  label: React.ReactNode,
  key: string,
  icon?: React.ReactNode,
  children?: MenuItem[],
): MenuItem => {
  return {
    key,
    icon,
    children,
    label,
  } as MenuItem;
};

const MainLayout = () => {
  const [collapsed, setCollapsed] = useState(false);
  const { token } = theme.useToken();
  const location = useLocation();
  
  // 根据当前路径确定选中的菜单项
  const getSelectedKey = () => {
    const path = location.pathname;
    if (path.startsWith('/services')) return 'services';
    if (path.startsWith('/dns')) return 'dns';
    if (path.startsWith('/settings')) return 'settings';
    return 'dashboard'; // 默认选中概览
  };

  // 定义菜单项数组
  const menuItems: MenuItem[] = [
    getItem(<Link to="/">概览</Link>, 'dashboard', <DashboardOutlined />),
    getItem(<Link to="/services">服务列表</Link>, 'services', <AppstoreOutlined />),
    getItem(<Link to="/dns">DNS配置</Link>, 'dns', <GlobalOutlined />),
    getItem(<Link to="/settings">系统设置</Link>, 'settings', <SettingOutlined />),
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header className="kong-header">
        <div className="logo">
          <CloudServerOutlined style={{ 
            fontSize: '24px', 
            color: '#40a9ff'
          }} />
          <div className="logo-text">Kong Discovery</div>
        </div>
      </Header>
      
      <Layout>
        <Sider 
          className="kong-sider"
          collapsible 
          collapsed={collapsed} 
          onCollapse={(value) => setCollapsed(value)}
          width={220}
          theme="dark"
        >
          <Menu
            mode="inline"
            theme="dark"
            defaultSelectedKeys={[getSelectedKey()]}
            style={{ 
              height: '100%', 
              borderRight: 0,
              backgroundColor: 'transparent' 
            }}
            items={menuItems}
          />
        </Sider>
        
        <Layout className="kong-content-layout">
          <Content className="kong-content">
            <Outlet />
          </Content>
        </Layout>
      </Layout>
    </Layout>
  );
};

export default MainLayout; 