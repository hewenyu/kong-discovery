import { useEffect, useState } from 'react';
import { Table, Card, Typography, Tag, Space, Button, Input, Dropdown, Menu } from 'antd';
import { useNavigate } from 'react-router-dom';
import { serviceApi } from '../api/client';
import type { ServiceInstanceResponse } from '../api/client';
import { SearchOutlined, ReloadOutlined, SettingOutlined, LinkOutlined, DownOutlined } from '@ant-design/icons';

const { Title, Text } = Typography;

interface Service {
  serviceName: string;
  instanceId: string;
  ip: string;
  port: number;
  status: string;
  lastHeartbeat: string;
  metadata?: Record<string, string>;
}

const ServiceList = () => {
  const [loading, setLoading] = useState(true);
  const [services, setServices] = useState<Service[]>([]);
  const [searchValue, setSearchValue] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    fetchServices();
  }, []);

  const fetchServices = async () => {
    setLoading(true);
    try {
      const response = await serviceApi.getAllServiceInstances();
      if (response.success && response.instances) {
        // 转换API响应为组件状态格式
        const serviceInstances = response.instances.map((instance: ServiceInstanceResponse) => ({
          serviceName: instance.service_name,
          instanceId: instance.instance_id,
          ip: instance.ip_address,
          port: instance.port,
          status: instance.status,
          lastHeartbeat: instance.last_heartbeat,
          metadata: instance.metadata
        }));
        setServices(serviceInstances);
      } else {
        setServices([]);
      }
    } catch (error) {
      console.error('获取服务列表失败:', error);
      setServices([]);
    } finally {
      setLoading(false);
    }
  };

  // 过滤服务列表
  const filteredServices = searchValue
    ? services.filter(service => 
        service.serviceName.toLowerCase().includes(searchValue.toLowerCase()) ||
        service.instanceId.toLowerCase().includes(searchValue.toLowerCase()) ||
        service.ip.includes(searchValue))
    : services;

  // 按服务名分组
  const groupedServices = filteredServices.reduce((acc, service) => {
    const { serviceName } = service;
    if (!acc[serviceName]) {
      acc[serviceName] = [];
    }
    acc[serviceName].push(service);
    return acc;
  }, {} as Record<string, Service[]>);

  // 服务操作下拉菜单
  const getServiceMenu = (serviceName: string) => (
    <Menu>
      <Menu.Item 
        key="dns-settings" 
        icon={<SettingOutlined />}
        onClick={() => navigate(`/services/${serviceName}/dns-settings`)}
      >
        DNS设置
      </Menu.Item>
      <Menu.Item 
        key="dns-associations" 
        icon={<LinkOutlined />}
        onClick={() => navigate(`/services/${serviceName}/dns-associations`)}
      >
        DNS关联关系
      </Menu.Item>
    </Menu>
  );

  const columns = [
    {
      title: '服务名称',
      dataIndex: 'serviceName',
      key: 'serviceName',
      render: (text: string, record: Service) => {
        // 检查是否是此服务的第一个实例，如果是，才显示服务操作菜单
        const isFirstInstance = groupedServices[text][0].instanceId === record.instanceId;
        
        return (
          <Space>
            <Text strong className="kong-text-accent">{text}</Text>
            {isFirstInstance && (
              <Dropdown overlay={getServiceMenu(text)}>
                <Button type="text" size="small">
                  服务操作 <DownOutlined />
                </Button>
              </Dropdown>
            )}
          </Space>
        );
      },
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
        <Tag color={status === 'active' ? 'success' : 'error'}>
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
            实例详情
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="kong-card">
        <Title level={3}>服务列表</Title>
        <Text type="secondary">管理已注册的服务及其实例</Text>
        
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 20, marginBottom: 20 }}>
          <Input 
            placeholder="搜索服务名称或IP" 
            style={{ width: 300 }} 
            value={searchValue}
            onChange={e => setSearchValue(e.target.value)}
            prefix={<SearchOutlined style={{ color: 'rgba(0,0,0,.25)' }} />}
            allowClear
          />
          
          <Button 
            type="primary" 
            icon={<ReloadOutlined />} 
            onClick={fetchServices}
            className="kong-button-primary"
          >
            刷新
          </Button>
        </div>
        
        <Table 
          className="kong-table"
          rowKey={(record) => `${record.serviceName}-${record.instanceId}`}
          columns={columns} 
          dataSource={filteredServices} 
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </div>
    </div>
  );
};

export default ServiceList; 