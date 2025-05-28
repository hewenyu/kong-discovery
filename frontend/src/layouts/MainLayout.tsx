import { useState } from 'react';
import { Layout, Menu, theme } from 'antd';
import { Link, Outlet } from 'react-router-dom';
import {
  ApiOutlined,
  AppstoreOutlined,
  SettingOutlined,
} from '@ant-design/icons';

const { Header, Content, Sider } = Layout;

const MainLayout = () => {
  const [collapsed, setCollapsed] = useState(false);
  const { token } = theme.useToken();

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
            defaultSelectedKeys={['services']}
            style={{ height: '100%', borderRight: 0 }}
          >
            <Menu.Item key="services" icon={<AppstoreOutlined />}>
              <Link to="/services">服务列表</Link>
            </Menu.Item>
            <Menu.Item key="dns" icon={<ApiOutlined />}>
              <Link to="/dns">DNS配置</Link>
            </Menu.Item>
            <Menu.Item key="settings" icon={<SettingOutlined />}>
              <Link to="/settings">系统设置</Link>
            </Menu.Item>
          </Menu>
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