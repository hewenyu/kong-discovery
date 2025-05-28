import { useState, useEffect } from 'react';
import { SettingOutlined, GlobalOutlined, SaveOutlined, ReloadOutlined } from '@ant-design/icons';
import { Typography, Form, Input, Button, Card, message, Spin, Space, Divider } from 'antd';
import { dnsApi } from '../api/client';
import type { DNSConfigResponse } from '../api/client';

const { Title, Text, Paragraph } = Typography;

interface DNSConfig {
  upstream_dns: string;
}

const DnsConfig = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState<DNSConfig>({ upstream_dns: '' });
  const [messageApi, contextHolder] = message.useMessage();

  // 获取DNS配置
  const fetchDNSConfig = async () => {
    setLoading(true);
    try {
      const response = await dnsApi.getDNSConfig();
      if (response.success && response.configs) {
        const dnsConfig: DNSConfig = {
          upstream_dns: response.configs.upstream_dns || '',
        };
        setConfig(dnsConfig);
        form.setFieldsValue(dnsConfig);
      }
    } catch (error) {
      console.error('获取DNS配置失败:', error);
      messageApi.error('获取DNS配置失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  // 更新DNS配置
  const updateDNSConfig = async (values: DNSConfig) => {
    setSaving(true);
    try {
      const response = await dnsApi.updateUpstreamDNS(values.upstream_dns);
      if (response.success) {
        messageApi.success('DNS配置更新成功');
        setConfig(values);
      } else {
        messageApi.error(response.message || '更新DNS配置失败');
      }
    } catch (error) {
      console.error('更新DNS配置失败:', error);
      messageApi.error('更新DNS配置失败，请稍后重试');
    } finally {
      setSaving(false);
    }
  };

  useEffect(() => {
    fetchDNSConfig();
  }, []);

  const handleSubmit = (values: DNSConfig) => {
    updateDNSConfig(values);
  };

  return (
    <div>
      {contextHolder}
      <div className="kong-card">
        <div style={{ marginBottom: 16 }}>
          <Title level={3} style={{ margin: 0 }}>DNS配置</Title>
          <Text type="secondary">管理DNS服务器配置和上游DNS服务器</Text>
        </div>
        
        {loading ? (
          <div style={{ display: 'flex', justifyContent: 'center', padding: '40px 0' }}>
            <Spin size="large" />
          </div>
        ) : (
          <Card 
            title={
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <GlobalOutlined style={{ marginRight: 8 }} />
                <span>上游DNS服务器配置</span>
              </div>
            }
            bordered={false}
            className="kong-inner-card"
          >
            <Paragraph>
              上游DNS服务器用于解析未知域名。当Kong Discovery无法从内部记录中解析域名时，查询会被转发到此服务器。
            </Paragraph>
            
            <Form
              form={form}
              layout="vertical"
              onFinish={handleSubmit}
              initialValues={config}
            >
              <Form.Item
                name="upstream_dns"
                label="上游DNS服务器地址"
                rules={[
                  { required: true, message: '请输入上游DNS服务器地址' },
                  { pattern: /^.+:\d+$/, message: '格式应为: IP地址或域名:端口，例如: 8.8.8.8:53' }
                ]}
              >
                <Input placeholder="输入格式: IP地址或域名:端口，例如: 8.8.8.8:53" />
              </Form.Item>
              
              <Form.Item>
                <Space>
                  <Button 
                    type="primary" 
                    htmlType="submit" 
                    icon={<SaveOutlined />} 
                    loading={saving}
                    className="kong-button-primary"
                  >
                    保存配置
                  </Button>
                  <Button 
                    icon={<ReloadOutlined />} 
                    onClick={fetchDNSConfig}
                    disabled={loading}
                  >
                    刷新
                  </Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>
        )}
      </div>
    </div>
  );
};

export default DnsConfig; 