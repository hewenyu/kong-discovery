import { useEffect, useState } from 'react';
import { Table, Card, Typography, Tag, Space, Button } from 'antd';
import { useNavigate } from 'react-router-dom';
import { serviceApi } from '../api/client';

const { Title } = Typography;

interface Service {
  serviceName: string;
  instanceId: string;
  ip: string;
  port: number;
  status: string;
  metadata?: Record<string, string>;
  registeredAt: string;
  lastHeartbeat: string;
}

const ServiceList = () => {
  const [loading, setLoading] = useState(true);
  const [services, setServices] = useState<Service[]>([]);
  const navigate = useNavigate();

  useEffect(() => {
    fetchServices();
  }, []);

  const fetchServices = async () => {
    setLoading(true);
    try {
      const data = await serviceApi.getServices();
      setServices(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('获取服务列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    {
      title: '服务名称',
      dataIndex: 'serviceName',
      key: 'serviceName',
    },
    {
      title: '实例ID',
      dataIndex: 'instanceId',
      key: 'instanceId',
      ellipsis: true,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
    },
    {
      title: '端口',
      dataIndex: 'port',
      key: 'port',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'red'}>
          {status === 'active' ? '活跃' : '离线'}
        </Tag>
      ),
    },
    {
      title: '最后心跳',
      dataIndex: 'lastHeartbeat',
      key: 'lastHeartbeat',
      render: (date: string) => new Date(date).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: Service) => (
        <Space size="middle">
          <Button 
            type="link" 
            onClick={() => navigate(`/services/${record.serviceName}/${record.instanceId}`)}
          >
            详情
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Title level={3}>服务列表</Title>
        
        <Space style={{ marginBottom: 16 }}>
          <Button type="primary" onClick={fetchServices}>
            刷新
          </Button>
        </Space>
        
        <Table 
          rowKey={(record) => `${record.serviceName}-${record.instanceId}`}
          columns={columns} 
          dataSource={services} 
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Space>
    </Card>
  );
};

export default ServiceList; 