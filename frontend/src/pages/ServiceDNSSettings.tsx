import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Button, Card, Select, InputNumber, Form, Input, Switch, message, Table, Space, Tag } from 'antd';
import { serviceApi, type ServiceDNSSettings as ServiceDNSSettingsType, type ServiceInstanceResponse } from '../api/client';

const { Option } = Select;

const ServiceDNSSettings: React.FC = () => {
  const { serviceName } = useParams<{ serviceName: string }>();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [saveLoading, setSaveLoading] = useState(false);
  const [settings, setSettings] = useState<ServiceDNSSettingsType | null>(null);
  const [form] = Form.useForm();
  const [useWeights, setUseWeights] = useState(false);
  const [instances, setInstances] = useState<ServiceInstanceResponse[]>([]);
  
  // 加载服务DNS设置
  const loadSettings = async () => {
    if (!serviceName) return;
    
    setLoading(true);
    try {
      const response = await serviceApi.getServiceDNSSettings(serviceName);
      if (response.success) {
        setSettings(response.settings);
        form.setFieldsValue(response.settings);
        setUseWeights(response.settings.load_balance_policy === 'weighted');
      } else {
        message.error(`加载DNS设置失败: ${response.message}`);
      }
    } catch (error) {
      console.error('加载DNS设置出错:', error);
      message.error('加载DNS设置失败，请检查网络连接');
    } finally {
      setLoading(false);
    }
  };
  
  // 加载服务实例
  const loadInstances = async () => {
    if (!serviceName) return;
    
    try {
      const response = await serviceApi.getAllServiceInstances();
      if (response.success) {
        setInstances(response.instances.filter(i => i.service_name === serviceName));
      } else {
        message.error(`加载服务实例失败: ${response.message}`);
      }
    } catch (error) {
      console.error('加载服务实例出错:', error);
      message.error('加载服务实例失败，请检查网络连接');
    }
  };
  
  // 保存设置
  const handleSave = async (values: any) => {
    if (!serviceName) return;
    
    setSaveLoading(true);
    try {
      // 处理权重数据
      if (values.load_balance_policy === 'weighted' && useWeights) {
        const instanceWeights: Record<string, number> = {};
        instances.forEach(instance => {
          const weight = values[`weight_${instance.instance_id}`];
          if (weight !== undefined) {
            instanceWeights[instance.instance_id] = weight;
          }
        });
        values.instance_weights = instanceWeights;
      } else {
        values.instance_weights = {};
      }
      
      // 删除临时字段
      instances.forEach(instance => {
        delete values[`weight_${instance.instance_id}`];
      });
      
      const response = await serviceApi.updateServiceDNSSettings(serviceName, values);
      if (response.success) {
        message.success('DNS设置保存成功');
        loadSettings(); // 重新加载设置
      } else {
        message.error(`保存DNS设置失败: ${response.message}`);
      }
    } catch (error) {
      console.error('保存DNS设置出错:', error);
      message.error('保存DNS设置失败，请检查网络连接');
    } finally {
      setSaveLoading(false);
    }
  };
  
  // 策略变更处理
  const handlePolicyChange = (value: string) => {
    setUseWeights(value === 'weighted');
  };
  
  useEffect(() => {
    if (serviceName) {
      loadSettings();
      loadInstances();
    } else {
      navigate('/services');
    }
  }, [serviceName]);
  
  // 初始化表单值
  useEffect(() => {
    if (settings) {
      form.setFieldsValue(settings);
      
      // 如果有权重配置，设置实例权重字段
      if (settings.instance_weights && Object.keys(settings.instance_weights).length > 0) {
        for (const [instanceId, weight] of Object.entries(settings.instance_weights)) {
          form.setFieldValue(`weight_${instanceId}`, weight);
        }
      }
    }
  }, [settings, instances]);
  
  return (
    <div>
      <Card 
        title={`服务 ${serviceName} DNS设置`}
        extra={<Button onClick={() => navigate('/services')}>返回服务列表</Button>}
        loading={loading}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={{
            load_balance_policy: 'round-robin',
            a_ttl: 60,
            srv_ttl: 60,
          }}
        >
          <Form.Item
            name="load_balance_policy"
            label="负载均衡策略"
            rules={[{ required: true, message: '请选择负载均衡策略' }]}
          >
            <Select onChange={handlePolicyChange}>
              <Option value="round-robin">轮询 (Round-Robin)</Option>
              <Option value="random">随机 (Random)</Option>
              <Option value="weighted">加权 (Weighted)</Option>
              <Option value="first-only">仅第一个实例 (First Only)</Option>
            </Select>
          </Form.Item>
          
          <Form.Item
            name="a_ttl"
            label="A记录TTL (秒)"
            rules={[{ required: true, message: '请输入A记录TTL' }]}
          >
            <InputNumber min={1} max={86400} style={{ width: '100%' }} />
          </Form.Item>
          
          <Form.Item
            name="srv_ttl"
            label="SRV记录TTL (秒)"
            rules={[{ required: true, message: '请输入SRV记录TTL' }]}
          >
            <InputNumber min={1} max={86400} style={{ width: '100%' }} />
          </Form.Item>
          
          <Form.Item
            name="custom_domain"
            label="自定义域名 (可选，留空使用默认域名)"
          >
            <Input placeholder="example.svc.cluster.local" />
          </Form.Item>
          
          {useWeights && instances.length > 0 && (
            <>
              <h3>实例权重配置</h3>
              <p>为每个实例设置权重值（1-100）。权重越高，被选中的概率越大。</p>
              
              {instances.map(instance => (
                <Form.Item
                  key={instance.instance_id}
                  name={`weight_${instance.instance_id}`}
                  label={`实例 ${instance.instance_id} (${instance.ip_address}:${instance.port})`}
                  initialValue={10}
                >
                  <InputNumber min={1} max={100} style={{ width: '100%' }} />
                </Form.Item>
              ))}
            </>
          )}
          
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={saveLoading}>
              保存设置
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default ServiceDNSSettings; 