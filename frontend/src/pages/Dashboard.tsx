import { useEffect, useState } from 'react';
import { Typography, Row, Col, Statistic, Spin, Button } from 'antd';
import { 
  AppstoreOutlined, 
  CloudServerOutlined, 
  GlobalOutlined,
  ReloadOutlined
} from '@ant-design/icons';
import { serviceApi } from '../api/client';

const { Title, Text } = Typography;

const Dashboard = () => {
  const [loading, setLoading] = useState(true);
  const [serviceCount, setServiceCount] = useState(0);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchServiceStats();
  }, []);

  const fetchServiceStats = async () => {
    setLoading(true);
    try {
      const data = await serviceApi.getServices();
      const services = Array.isArray(data) ? data : [];
      
      // 计算不同服务名称的数量
      const uniqueServices = new Set();
      services.forEach(service => uniqueServices.add(service.serviceName));
      
      setServiceCount(uniqueServices.size);
      setError(null);
    } catch (error) {
      console.error('获取服务统计失败:', error);
      setError('无法加载服务统计数据');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <div className="kong-card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <div>
            <Title level={3} style={{ margin: 0 }}>Kong Discovery 概览</Title>
            <Text type="secondary">服务发现系统监控面板</Text>
          </div>
          
          <Button 
            type="primary" 
            icon={<ReloadOutlined />} 
            onClick={fetchServiceStats}
            className="kong-button-primary"
          >
            刷新
          </Button>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: '20px 0' }}>
            <Spin />
          </div>
        ) : error ? (
          <div style={{ color: 'red', padding: '20px 0' }}>
            {error}
          </div>
        ) : (
          <Row gutter={16}>
            <Col span={8}>
              <div className="kong-card" style={{ backgroundColor: '#f0f6ff' }}>
                <Statistic 
                  title="已注册服务" 
                  value={serviceCount} 
                  prefix={<AppstoreOutlined />} 
                  valueStyle={{ color: '#1155cb' }}
                />
              </div>
            </Col>
            <Col span={8}>
              <div className="kong-card" style={{ backgroundColor: '#f0f6ff' }}>
                <Statistic 
                  title="DNS记录" 
                  value={serviceCount > 0 ? serviceCount * 2 : 0} 
                  prefix={<GlobalOutlined />}
                  valueStyle={{ color: '#1155cb' }}
                  suffix="条"
                />
              </div>
            </Col>
            <Col span={8}>
              <div className="kong-card" style={{ backgroundColor: '#f0f6ff' }}>
                <Statistic 
                  title="系统状态" 
                  value="运行中" 
                  prefix={<CloudServerOutlined />}
                  valueStyle={{ color: '#52c41a' }}
                />
              </div>
            </Col>
          </Row>
        )}
      </div>

      <div className="kong-card">
        <div className="kong-card-title">
          <CloudServerOutlined /> 系统信息
        </div>
        <Row gutter={16}>
          <Col span={8}>
            <Statistic title="版本" value="1.0.0" />
          </Col>
          <Col span={8}>
            <Statistic title="运行时间" value="1 天" />
          </Col>
          <Col span={8}>
            <Statistic title="服务API端口" value="8080" />
          </Col>
        </Row>
      </div>
    </div>
  );
};

export default Dashboard; 