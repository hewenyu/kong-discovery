import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Descriptions, Button, Spin, Typography, Space, Tag, Divider } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { serviceApi } from '../api/client';

const { Title } = Typography;

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
      <Card>
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
      </Card>
    );
  }

  return (
    <Card>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Space>
          <Button 
            type="link" 
            icon={<ArrowLeftOutlined />} 
            onClick={() => navigate('/services')}
            style={{ padding: 0 }}
          >
            返回服务列表
          </Button>
        </Space>
        
        <Title level={3}>{service.serviceName} 服务详情</Title>
        
        <Descriptions bordered column={2}>
          <Descriptions.Item label="服务名称">{service.serviceName}</Descriptions.Item>
          <Descriptions.Item label="实例ID">{service.instanceId}</Descriptions.Item>
          <Descriptions.Item label="IP地址">{service.ip}</Descriptions.Item>
          <Descriptions.Item label="端口">{service.port}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={service.status === 'active' ? 'green' : 'red'}>
              {service.status === 'active' ? '活跃' : '离线'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="注册时间">
            {new Date(service.registeredAt).toLocaleString()}
          </Descriptions.Item>
          <Descriptions.Item label="最后心跳">
            {new Date(service.lastHeartbeat).toLocaleString()}
          </Descriptions.Item>
        </Descriptions>
        
        {service.metadata && Object.keys(service.metadata).length > 0 && (
          <>
            <Divider orientation="left">元数据</Divider>
            <Descriptions bordered column={1}>
              {Object.entries(service.metadata).map(([key, value]) => (
                <Descriptions.Item key={key} label={key}>
                  {value}
                </Descriptions.Item>
              ))}
            </Descriptions>
          </>
        )}
        
        {service.dnsRecords && service.dnsRecords.length > 0 && (
          <>
            <Divider orientation="left">DNS记录</Divider>
            <Descriptions bordered column={1}>
              {service.dnsRecords.map((record, index) => (
                <Descriptions.Item key={index} label={`${record.type} 记录 (${record.name})`}>
                  {record.value}
                </Descriptions.Item>
              ))}
            </Descriptions>
          </>
        )}
      </Space>
    </Card>
  );
};

export default ServiceDetailPage; 