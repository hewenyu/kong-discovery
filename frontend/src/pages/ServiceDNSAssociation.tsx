import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Button, Card, Table, Space, message, Modal, Form, Input, Select, Tag, Popconfirm } from 'antd';
import { PlusOutlined, DeleteOutlined, LinkOutlined } from '@ant-design/icons';
import { serviceApi, dnsApi, type ServiceDNSAssociationResponse } from '../api/client';

const { Option } = Select;

const ServiceDNSAssociation: React.FC = () => {
  const { serviceName } = useParams<{ serviceName: string }>();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [associations, setAssociations] = useState<Record<string, string[]>>({});
  const [modalVisible, setModalVisible] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [form] = Form.useForm();
  const [domains, setDomains] = useState<string[]>([]);
  const [domainsLoading, setDomainsLoading] = useState(false);
  
  // 加载服务DNS关联关系
  const loadAssociations = async () => {
    if (!serviceName) return;
    
    setLoading(true);
    try {
      const response = await serviceApi.getServiceDNSAssociations(serviceName);
      if (response.success) {
        setAssociations(response.associations || {});
      } else {
        message.error(`加载DNS关联关系失败: ${response.message}`);
      }
    } catch (error) {
      console.error('加载DNS关联关系出错:', error);
      message.error('加载DNS关联关系失败，请检查网络连接');
    } finally {
      setLoading(false);
    }
  };
  
  // 加载所有DNS域名
  const loadDomains = async () => {
    setDomainsLoading(true);
    try {
      const response = await dnsApi.getAllDNSDomains();
      if (response.success) {
        setDomains(response.domains || []);
      } else {
        message.error(`加载DNS域名列表失败: ${response.message}`);
      }
    } catch (error) {
      console.error('加载DNS域名列表出错:', error);
      message.error('加载DNS域名列表失败，请检查网络连接');
    } finally {
      setDomainsLoading(false);
    }
  };
  
  // 创建关联
  const handleCreateAssociation = async (values: any) => {
    if (!serviceName) return;
    
    setModalLoading(true);
    try {
      const response = await serviceApi.associateDNSWithService(
        serviceName, 
        values.domain, 
        values.record_type
      );
      
      if (response.success) {
        message.success('DNS关联创建成功');
        setModalVisible(false);
        form.resetFields();
        loadAssociations();
      } else {
        message.error(`创建DNS关联失败: ${response.message}`);
      }
    } catch (error) {
      console.error('创建DNS关联出错:', error);
      message.error('创建DNS关联失败，请检查网络连接');
    } finally {
      setModalLoading(false);
    }
  };
  
  // 删除关联
  const handleDeleteAssociation = async (domain: string, recordType: string) => {
    if (!serviceName) return;
    
    try {
      const response = await serviceApi.disassociateDNSFromService(
        serviceName,
        domain,
        recordType
      );
      
      if (response.success) {
        message.success('DNS关联删除成功');
        loadAssociations();
      } else {
        message.error(`删除DNS关联失败: ${response.message}`);
      }
    } catch (error) {
      console.error('删除DNS关联出错:', error);
      message.error('删除DNS关联失败，请检查网络连接');
    }
  };
  
  useEffect(() => {
    if (serviceName) {
      loadAssociations();
    } else {
      navigate('/services');
    }
  }, [serviceName]);
  
  useEffect(() => {
    if (modalVisible) {
      loadDomains();
    }
  }, [modalVisible]);
  
  // 构建表格数据
  const tableData = Object.entries(associations).flatMap(([domain, recordTypes]) =>
    recordTypes.map((recordType, index) => ({
      key: `${domain}-${recordType}`,
      domain,
      record_type: recordType,
      is_first_of_domain: index === 0,
      domain_row_span: index === 0 ? recordTypes.length : 0,
    }))
  );
  
  // 表格列定义
  const columns = [
    {
      title: '域名',
      dataIndex: 'domain',
      key: 'domain',
      render: (text: string, record: any) => ({
        children: <a href={`/dns/records?domain=${text}`} target="_blank">{text}</a>,
        props: {
          rowSpan: record.domain_row_span,
        },
      }),
    },
    {
      title: '记录类型',
      dataIndex: 'record_type',
      key: 'record_type',
      render: (text: string) => <Tag color="blue">{text}</Tag>,
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: any) => (
        <Space size="middle">
          <Popconfirm
            title="确定要删除这个DNS关联关系吗?"
            onConfirm={() => handleDeleteAssociation(record.domain, record.record_type)}
            okText="确定"
            cancelText="取消"
          >
            <Button 
              type="text" 
              danger 
              icon={<DeleteOutlined />}
            >
              删除关联
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];
  
  return (
    <div>
      <Card 
        title={`服务 ${serviceName} 的DNS关联关系`}
        extra={
          <Space>
            <Button onClick={() => navigate('/services')}>返回服务列表</Button>
            <Button 
              type="primary" 
              icon={<PlusOutlined />}
              onClick={() => setModalVisible(true)}
            >
              添加关联
            </Button>
          </Space>
        }
        loading={loading}
      >
        <Table 
          dataSource={tableData} 
          columns={columns} 
          pagination={false}
          locale={{ emptyText: '暂无DNS关联关系' }}
        />
      </Card>
      
      {/* 添加关联Modal */}
      <Modal
        title="添加DNS关联关系"
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateAssociation}
        >
          <Form.Item
            name="domain"
            label="域名"
            rules={[{ required: true, message: '请选择域名' }]}
          >
            <Select 
              loading={domainsLoading}
              placeholder="选择一个域名"
              showSearch
              filterOption={(input, option) =>
                (option?.children as unknown as string).toLowerCase().indexOf(input.toLowerCase()) >= 0
              }
              dropdownRender={menu => (
                <div>
                  {menu}
                  <div style={{ padding: 8, borderTop: '1px solid #e8e8e8' }}>
                    <Button 
                      type="link" 
                      icon={<PlusOutlined />} 
                      onClick={() => navigate('/dns/records')}
                    >
                      添加新域名
                    </Button>
                  </div>
                </div>
              )}
            >
              {domains.map(domain => (
                <Option key={domain} value={domain}>{domain}</Option>
              ))}
            </Select>
          </Form.Item>
          
          <Form.Item
            name="record_type"
            label="记录类型"
            rules={[{ required: true, message: '请选择记录类型' }]}
          >
            <Select placeholder="选择记录类型">
              <Option value="A">A</Option>
              <Option value="SRV">SRV</Option>
              <Option value="CNAME">CNAME</Option>
              <Option value="TXT">TXT</Option>
            </Select>
          </Form.Item>
          
          <Form.Item>
            <Space>
              <Button 
                type="primary" 
                htmlType="submit" 
                loading={modalLoading}
                icon={<LinkOutlined />}
              >
                创建关联
              </Button>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ServiceDNSAssociation; 