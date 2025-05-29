import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Descriptions, Button, Spin, Typography, Space, Tag, Divider, Row, Col } from 'antd';
import { ArrowLeftOutlined, ClockCircleOutlined, CodeOutlined, InfoCircleOutlined, GlobalOutlined } from '@ant-design/icons';
import { serviceApi } from '../api/client';
import type { ServiceDetailResponse } from '../api/client';

const { Title, Text } = Typography;

const ServiceDetailPage = () => {
  const { serviceName, instanceId } = useParams<{ serviceName: string; instanceId: string }>();
  const [loading, setLoading] = useState(true);
  const [service, setService] = useState<ServiceDetailResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (serviceName && instanceId) {
      fetchServiceDetail(serviceName, instanceId);
    }
  }, [serviceName, instanceId]);

  const fetchServiceDetail = async (name: string, id: string) => {
    setLoading(true);
    setError(null);
    try {
      const response = await serviceApi.getServiceDetail(name, id);
      if (response.success) {
        setService(response);
      } else {
        setError(response.message || '获取服务详情失败');
        setService(null);
      }
    } catch (err) {
      console.error('获取服务详情失败:', err);
      setError('获取服务详情失败，请稍后重试');
      setService(null);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px 0' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (error || !service) {
    return (
      <div className="kong-card">
        <Space direction="vertical" style={{ width: '100%' }}>
          <Button 
            type="link" 
            icon={<ArrowLeftOutlined />} 
            onClick={() => navigate('/services')}
            style={{ padding: 0 }}
          >
            返回服务列表
          </Button>
          <Title level={4} style={{ color: 'red' }}>{error || '服务不存在'}</Title>
        </Space>
      </div>
    );
  }

  return (
    <div>
      <div className="kong-card">
        <Button 
          type="link" 
          icon={<ArrowLeftOutlined />} 
          onClick={() => navigate('/services')}
          style={{ padding: 0, marginBottom: 16 }}
        >
          返回服务列表
        </Button>
        
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <Title level={3} style={{ margin: 0 }}>
            {service.service_name} 
            <Tag color="success" style={{ marginLeft: 8 }}>
              活跃
            </Tag>
          </Title>
          
          <Button 
            type="primary" 
            onClick={() => fetchServiceDetail(service.service_name, service.instance_id)}
            className="kong-button-primary"
          >
            刷新
          </Button>
        </div>
        
        <Text type="secondary">实例ID: {service.instance_id}</Text>
      </div>

      <Row gutter={16}>
        <Col span={12}>
          <div className="kong-card">
            <div className="kong-card-title">
              <InfoCircleOutlined /> 基本信息
            </div>
            <Descriptions bordered column={1} size="small">
              <Descriptions.Item label="服务名称">{service.service_name}</Descriptions.Item>
              <Descriptions.Item label="实例ID">{service.instance_id}</Descriptions.Item>
              <Descriptions.Item label="IP地址">{service.ip_address}</Descriptions.Item>
              <Descriptions.Item label="端口">{service.port}</Descriptions.Item>
              <Descriptions.Item label="TTL">{service.ttl}秒</Descriptions.Item>
            </Descriptions>
          </div>
        </Col>
        
        <Col span={12}>
          <div className="kong-card">
            <div className="kong-card-title">
              <ClockCircleOutlined /> 时间信息
            </div>
            <Descriptions bordered column={1} size="small">
              <Descriptions.Item label="最后心跳">
                {service.last_heartbeat ? new Date(service.last_heartbeat).toLocaleString() : '未知'}
              </Descriptions.Item>
              <Descriptions.Item label="数据刷新时间">
                {new Date(service.timestamp).toLocaleString()}
              </Descriptions.Item>
            </Descriptions>
          </div>
        </Col>
      </Row>
      
      {service.metadata && Object.keys(service.metadata).length > 0 && (
        <div className="kong-card">
          <div className="kong-card-title">
            <CodeOutlined /> 元数据
          </div>
          <Descriptions bordered column={2} size="small">
            {Object.entries(service.metadata).map(([key, value]) => (
              <Descriptions.Item key={key} label={key}>
                {value}
              </Descriptions.Item>
            ))}
          </Descriptions>
        </div>
      )}
      
      {service.metadata?.domain && (
        <div className="kong-card">
          <div className="kong-card-title">
            <GlobalOutlined /> DNS信息
          </div>
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="域名">
              {service.metadata.domain}
            </Descriptions.Item>
            <Descriptions.Item label="A记录">
              {service.ip_address}
            </Descriptions.Item>
            <Descriptions.Item label="SRV记录">
              {`10 10 ${service.port} ${service.instance_id}.${service.metadata.domain}`}
            </Descriptions.Item>
          </Descriptions>
        </div>
      )}
    </div>
  );
};

export default ServiceDetailPage; 