import { useState, useEffect } from 'react';
import { Table, Button, Input, Select, Modal, Form, InputNumber, Tag, message, Card, Typography, Divider, Space, Spin } from 'antd';
import { PlusOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { dnsApi } from '../api/client';
import type { DNSRecord } from '../api/client';

const { Title, Text } = Typography;
const { Option } = Select;

const DnsRecords: React.FC = () => {
  // 状态管理
  const [loading, setLoading] = useState<boolean>(false);
  const [domains, setDomains] = useState<string[]>([]);
  const [selectedDomain, setSelectedDomain] = useState<string>('');
  const [records, setRecords] = useState<Record<string, DNSRecord>>({});
  const [isModalVisible, setIsModalVisible] = useState<boolean>(false);
  
  // 表单实例
  const [form] = Form.useForm();

  // 获取所有DNS域名
  const fetchDomains = async () => {
    try {
      setLoading(true);
      const response = await dnsApi.getAllDNSDomains();
      if (response.success) {
        setDomains(response.domains);
        // 如果有域名且没有选择，则选择第一个
        if (response.domains.length > 0 && !selectedDomain) {
          setSelectedDomain(response.domains[0]);
        }
      } else {
        message.error(response.message || '获取DNS域名失败');
      }
    } catch (error) {
      message.error('获取DNS域名列表失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 获取选定域名的所有DNS记录
  const fetchRecords = async () => {
    if (!selectedDomain) return;
    
    try {
      setLoading(true);
      const response = await dnsApi.getDNSRecords(selectedDomain);
      if (response.success) {
        setRecords(response.records);
      } else {
        setRecords({});
        message.error(response.message || '获取DNS记录失败');
      }
    } catch (error) {
      setRecords({});
      message.error('获取DNS记录失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 初始加载
  useEffect(() => {
    fetchDomains();
  }, []);

  // 当选择域名变化时，加载该域名的DNS记录
  useEffect(() => {
    if (selectedDomain) {
      fetchRecords();
    }
  }, [selectedDomain]);

  // 创建DNS记录
  const handleCreateRecord = async (values: any) => {
    try {
      setLoading(true);
      const { domain, type, value, ttl, tags } = values;
      const tagsArray = tags ? tags.split(',').map((tag: string) => tag.trim()) : undefined;
      
      const response = await dnsApi.createDNSRecord(domain, type, value, ttl, tagsArray);
      if (response.success) {
        message.success('DNS记录创建成功');
        setIsModalVisible(false);
        form.resetFields();
        // 如果创建的是当前选中域名的记录，刷新记录列表
        if (domain === selectedDomain) {
          fetchRecords();
        } else {
          // 如果是新域名，刷新域名列表并选择该域名
          fetchDomains();
          setSelectedDomain(domain);
        }
      } else {
        message.error(response.message || '创建DNS记录失败');
      }
    } catch (error) {
      message.error('创建DNS记录失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 删除DNS记录
  const handleDeleteRecord = async (type: string) => {
    try {
      setLoading(true);
      const response = await dnsApi.deleteDNSRecord(selectedDomain, type);
      if (response.success) {
        message.success('DNS记录删除成功');
        fetchRecords(); // 刷新记录列表
      } else {
        message.error(response.message || '删除DNS记录失败');
      }
    } catch (error) {
      message.error('删除DNS记录失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 刷新列表
  const handleRefresh = () => {
    fetchDomains();
    if (selectedDomain) {
      fetchRecords();
    }
  };

  // 表格列定义
  const columns = [
    {
      title: '记录类型',
      dataIndex: 'type',
      key: 'type',
    },
    {
      title: '记录值',
      dataIndex: 'value',
      key: 'value',
    },
    {
      title: 'TTL(秒)',
      dataIndex: 'ttl',
      key: 'ttl',
    },
    {
      title: '标签',
      dataIndex: 'tags',
      key: 'tags',
      render: (tags: string[]) => (
        tags ? (
          <span>
            {tags.map(tag => (
              <Tag color="blue" key={tag}>
                {tag}
              </Tag>
            ))}
          </span>
        ) : null
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: DNSRecord) => (
        <Button 
          type="primary" 
          danger 
          icon={<DeleteOutlined />}
          onClick={() => handleDeleteRecord(record.type)}
        >
          删除
        </Button>
      ),
    },
  ];

  // 准备表格数据
  const tableData = Object.entries(records).map(([recordType, record]) => ({
    key: recordType,
    type: recordType,
    value: record.value,
    ttl: record.ttl,
    tags: record.tags
  }));

  return (
    <div>
      <Title level={2}>DNS记录管理</Title>
      <Text type="secondary">管理自定义DNS记录，支持A、AAAA、CNAME、MX等记录类型</Text>

      <Divider />

      <Card>
        <Space style={{ marginBottom: 16 }}>
          <Select
            style={{ width: 240 }}
            placeholder="选择域名"
            value={selectedDomain}
            onChange={setSelectedDomain}
            disabled={loading}
          >
            {domains.map(domain => (
              <Option key={domain} value={domain}>{domain}</Option>
            ))}
          </Select>

          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setIsModalVisible(true)}
            disabled={loading}
          >
            添加记录
          </Button>

          <Button
            icon={<ReloadOutlined />}
            onClick={handleRefresh}
            disabled={loading}
          >
            刷新
          </Button>
        </Space>

        <Spin spinning={loading}>
          <Table
            columns={columns}
            dataSource={tableData}
            pagination={false}
            locale={{ emptyText: selectedDomain ? '没有找到DNS记录' : '请选择域名' }}
          />
        </Spin>
      </Card>

      {/* 添加DNS记录的模态框 */}
      <Modal
        title="添加DNS记录"
        open={isModalVisible}
        onCancel={() => setIsModalVisible(false)}
        footer={null}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateRecord}
        >
          <Form.Item
            name="domain"
            label="域名"
            rules={[{ required: true, message: '请输入域名' }]}
          >
            <Input placeholder="输入域名，如example.com" />
          </Form.Item>

          <Form.Item
            name="type"
            label="记录类型"
            rules={[{ required: true, message: '请选择记录类型' }]}
          >
            <Select placeholder="选择记录类型">
              <Option value="A">A - IPv4地址</Option>
              <Option value="AAAA">AAAA - IPv6地址</Option>
              <Option value="CNAME">CNAME - 别名</Option>
              <Option value="MX">MX - 邮件交换</Option>
              <Option value="TXT">TXT - 文本</Option>
              <Option value="SRV">SRV - 服务定位</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="value"
            label="记录值"
            rules={[{ required: true, message: '请输入记录值' }]}
          >
            <Input placeholder="根据记录类型输入对应的值" />
          </Form.Item>

          <Form.Item
            name="ttl"
            label="TTL(秒)"
            rules={[{ required: true, message: '请输入TTL' }]}
            initialValue={300}
          >
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="tags"
            label="标签(可选，逗号分隔)"
          >
            <Input placeholder="tag1,tag2,tag3" />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} style={{ marginRight: 8 }}>
              添加
            </Button>
            <Button onClick={() => setIsModalVisible(false)}>
              取消
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DnsRecords; 