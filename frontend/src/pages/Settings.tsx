import { SettingOutlined, ToolOutlined } from '@ant-design/icons';
import { Typography, Empty, Button } from 'antd';

const { Title, Text } = Typography;

const Settings = () => {
  return (
    <div className="kong-card">
      <div style={{ marginBottom: 16 }}>
        <Title level={3} style={{ margin: 0 }}>系统设置</Title>
        <Text type="secondary">管理Kong Discovery的系统配置</Text>
      </div>

      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', padding: '40px 0' }}>
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <span>
              系统设置功能正在开发中，即将推出
            </span>
          }
        >
          <Button type="primary" className="kong-button-primary" disabled>
            敬请期待
          </Button>
        </Empty>
      </div>
    </div>
  );
};

export default Settings; 