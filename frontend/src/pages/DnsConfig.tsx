import { useState, useEffect } from 'react';
import { SettingOutlined, GlobalOutlined, SaveOutlined, ReloadOutlined, PlusOutlined, MinusCircleOutlined, EditOutlined } from '@ant-design/icons';
import { Form, Input, Button, Card, message, Spin, Space, Typography, Divider, Row, Col } from 'antd';
import { dnsApi } from '../api/client';
import type { DNSConfigResponse } from '../api/client';
import { Link } from 'react-router-dom';

const { Title, Text } = Typography;

interface DNSConfig {
  upstream_dns: string[];
}

const DnsConfig = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState<DNSConfig>({ upstream_dns: [''] });
  const [messageApi, contextHolder] = message.useMessage();

  // 获取DNS配置
  const fetchDNSConfig = async () => {
    setLoading(true);
    try {
      const response = await dnsApi.getDNSConfig();
      if (response.success && response.configs) {
        const dnsConfig: DNSConfig = {
          upstream_dns: response.configs.upstream_dns || ['8.8.8.8:53']
        };
        setConfig(dnsConfig);
        form.setFieldsValue(dnsConfig);
      } else {
        messageApi.error(response.message || '获取DNS配置失败');
      }
    } catch (error) {
      console.error('获取DNS配置出错:', error);
      messageApi.error('获取DNS配置失败');
    } finally {
      setLoading(false);
    }
  };

  // 更新DNS配置
  const updateDNSConfig = async (values: DNSConfig) => {
    setSaving(true);
    try {
      // 过滤掉空的上游DNS
      const validUpstreamDNS = values.upstream_dns.filter(dns => dns.trim() !== '');
      
      if (validUpstreamDNS.length === 0) {
        messageApi.error('至少需要一个上游DNS服务器');
        setSaving(false);
        return;
      }
      
      const response = await dnsApi.updateUpstreamDNS(validUpstreamDNS);
      if (response.success) {
        messageApi.success('DNS配置更新成功');
        // 更新本地状态
        setConfig({
          upstream_dns: response.configs.upstream_dns
        });
      } else {
        messageApi.error(response.message || 'DNS配置更新失败');
      }
    } catch (error) {
      console.error('更新DNS配置出错:', error);
      messageApi.error('DNS配置更新失败');
    } finally {
      setSaving(false);
    }
  };

  // 添加一个上游DNS输入框
  const addUpstreamDNS = () => {
    const upstreamDNS = form.getFieldValue('upstream_dns') || [];
    form.setFieldsValue({
      upstream_dns: [...upstreamDNS, '']
    });
  };

  // 移除上游DNS输入框
  const removeUpstreamDNS = (index: number) => {
    const upstreamDNS = form.getFieldValue('upstream_dns') || [];
    if (upstreamDNS.length <= 1) {
      messageApi.warning('至少需要保留一个上游DNS服务器');
      return;
    }
    form.setFieldsValue({
      upstream_dns: upstreamDNS.filter((_: any, i: number) => i !== index)
    });
  };

  // 提交表单
  const handleSubmit = (values: DNSConfig) => {
    updateDNSConfig(values);
  };

  // 组件加载时获取配置
  useEffect(() => {
    fetchDNSConfig();
  }, []);

  return (
    <div>
      {contextHolder}
      <Row justify="space-between" align="middle">
        <Col>
          <Title level={2}><SettingOutlined /> DNS配置管理</Title>
          <Text type="secondary">配置上游DNS服务器和自定义DNS记录</Text>
        </Col>
        <Col>
          <Link to="/dns/records">
            <Button type="primary" icon={<EditOutlined />}>
              管理DNS记录
            </Button>
          </Link>
        </Col>
      </Row>

      <Divider />

      <Card title="上游DNS服务器配置" extra={
        <Button 
          type="primary" 
          icon={<ReloadOutlined />} 
          onClick={fetchDNSConfig} 
          loading={loading}
        >
          刷新
        </Button>
      }>
        <Spin spinning={loading}>
          <Form
            form={form}
            layout="vertical"
            onFinish={handleSubmit}
            initialValues={config}
          >
            <Form.Item
              label="上游DNS服务器"
              required
              extra="当本地无法解析域名时，查询请求将转发到这些上游DNS服务器。可以添加多个服务器进行负载均衡。"
            >
              <Form.List name="upstream_dns">
                {(fields) => (
                  <>
                    {fields.map((field, index) => (
                      <Space key={field.key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                        <Form.Item
                          {...field}
                          rules={[{ required: true, message: '请输入上游DNS服务器地址' }]}
                          style={{ marginBottom: 0 }}
                        >
                          <Input 
                            prefix={<GlobalOutlined />} 
                            placeholder="例如: 8.8.8.8:53" 
                            style={{ width: 300 }}
                          />
                        </Form.Item>
                        {fields.length > 1 ? (
                          <MinusCircleOutlined
                            className="dynamic-delete-button"
                            onClick={() => removeUpstreamDNS(index)}
                          />
                        ) : null}
                      </Space>
                    ))}
                    <Form.Item>
                      <Button
                        type="dashed"
                        onClick={addUpstreamDNS}
                        icon={<PlusOutlined />}
                      >
                        添加上游DNS服务器
                      </Button>
                    </Form.Item>
                  </>
                )}
              </Form.List>
            </Form.Item>

            <Form.Item>
              <Button 
                type="primary" 
                htmlType="submit" 
                icon={<SaveOutlined />} 
                loading={saving}
              >
                保存配置
              </Button>
            </Form.Item>
          </Form>
        </Spin>
      </Card>

      <Divider />

      <Card title="自定义DNS记录">
        <Space direction="vertical" style={{ width: '100%' }}>
          <Text>
            通过自定义DNS记录，您可以为特定域名配置解析规则，例如将特定域名指向内部服务IP。
          </Text>
          <Link to="/dns/records">
            <Button type="primary" icon={<EditOutlined />}>
              管理DNS记录
            </Button>
          </Link>
        </Space>
      </Card>
    </div>
  );
};

export default DnsConfig; 