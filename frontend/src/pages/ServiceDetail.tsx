import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Descriptions, Button, Spin, Typography, Space, Tag, Divider, Row, Col } from 'antd';
import { ArrowLeftOutlined, ClockCircleOutlined, CodeOutlined, InfoCircleOutlined, GlobalOutlined } from '@ant-design/icons';
import { serviceApi } from '../api/client';

const { Title, Text } = Typography;

interface ServiceDetail {
  serviceName: string;
  instanceId: string;
  ip: string;
  port: number;
  status: string;
  metadata?: Record<string, string>;
  registeredAt: string;
  lastHeartbeat: string;
  dnsRecords?: {
    type: string;
    name: string;
    value: string;
  }[];
}

const ServiceDetailPage = () => {
  const { serviceName, instanceId } = useParams<{ serviceName: string; instanceId: string }>();
  const [loading, setLoading] = useState(true);
  const [service, setService] = useState<ServiceDetail | null>(null);
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
      const data = await serviceApi.getServiceDetail(name, id);
      setService(data);
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
            {service.serviceName} 
            <Tag color={service.status === 'active' ? 'success' : 'error'} style={{ marginLeft: 8 }}>
              {service.status === 'active' ? '活跃' : '离线'}
            </Tag>
          </Title>
          
          <Button 
            type="primary" 
            onClick={() => fetchServiceDetail(service.serviceName, service.instanceId)}
            className="kong-button-primary"
          >
            刷新
          </Button>
        </div>
        
        <Text type="secondary">实例ID: {service.instanceId}</Text>
      </div>

      <Row gutter={16}>
        <Col span={12}>
          <div className="kong-card">
            <div className="kong-card-title">
              <InfoCircleOutlined /> 基本信息
            </div>
            <Descriptions bordered column={1} size="small">
              <Descriptions.Item label="服务名称">{service.serviceName}</Descriptions.Item>
              <Descriptions.Item label="实例ID">{service.instanceId}</Descriptions.Item>
              <Descriptions.Item label="IP地址">{service.ip}</Descriptions.Item>
              <Descriptions.Item label="端口">{service.port}</Descriptions.Item>
            </Descriptions>
          </div>
        </Col>
        
        <Col span={12}>
          <div className="kong-card">
            <div className="kong-card-title">
              <ClockCircleOutlined /> 时间信息
            </div>
            <Descriptions bordered column={1} size="small">
              <Descriptions.Item label="注册时间">
                {new Date(service.registeredAt).toLocaleString()}
              </Descriptions.Item>
              <Descriptions.Item label="最后心跳">
                {new Date(service.lastHeartbeat).toLocaleString()}
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
      
      {service.dnsRecords && service.dnsRecords.length > 0 && (
        <div className="kong-card">
          <div className="kong-card-title">
            <GlobalOutlined /> DNS记录
          </div>
          <Descriptions bordered column={1} size="small">
            {service.dnsRecords.map((record, index) => (
              <Descriptions.Item key={index} label={`${record.type} 记录 (${record.name})`}>
                {record.value}
              </Descriptions.Item>
            ))}
          </Descriptions>
        </div>
      )}
    </div>
  );
};

export default ServiceDetailPage; 