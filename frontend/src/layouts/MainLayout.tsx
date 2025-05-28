import { useState } from 'react';
import { Layout, Menu, theme } from 'antd';
import { Link, Outlet, useLocation } from 'react-router-dom';
import {
  ApiOutlined,
  AppstoreOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';

const { Header, Content, Sider } = Layout;

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
    return 'services'; // 默认选中服务列表
  };

  // 定义菜单项数组
  const menuItems: MenuItem[] = [
    getItem(<Link to="/services">服务列表</Link>, 'services', <AppstoreOutlined />),
    getItem(<Link to="/dns">DNS配置</Link>, 'dns', <ApiOutlined />),
    getItem(<Link to="/settings">系统设置</Link>, 'settings', <SettingOutlined />),
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ 
        display: 'flex', 
        alignItems: 'center', 
        background: token.colorPrimary, 
        color: 'white',
        padding: '0 16px'
      }}>
        <h1 style={{ margin: 0 }}>Kong Discovery</h1>
      </Header>
      
      <Layout>
        <Sider 
          collapsible 
          collapsed={collapsed} 
          onCollapse={(value) => setCollapsed(value)}
          width={200}
        >
          <Menu
            mode="inline"
            theme="dark"
            defaultSelectedKeys={[getSelectedKey()]}
            style={{ height: '100%', borderRight: 0 }}
            items={menuItems}
          />
        </Sider>
        
        <Layout style={{ padding: '0 24px 24px' }}>
          <Content
            style={{
              padding: 24,
              margin: '16px 0',
              background: token.colorBgContainer,
              borderRadius: token.borderRadiusLG,
              minHeight: 280,
            }}
          >
            <Outlet />
          </Content>
        </Layout>
      </Layout>
    </Layout>
  );
};

export default MainLayout; 