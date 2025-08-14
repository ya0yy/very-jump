import React, { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Typography,
  Space,
  Modal,
  Form,
  Input,
  Select,
  message,
  Popconfirm,
  Tag,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  KeyOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../stores/authStore';
import { credentialAPI } from '../services/api';
import type { Credential, CredentialCreateRequest } from '../types';

const { Title } = Typography;
const { Option } = Select;

const Credentials: React.FC = () => {
  const { user } = useAuthStore();
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingCredential, setEditingCredential] = useState<Credential | null>(null);
  const [form] = Form.useForm();

  useEffect(() => {
    fetchCredentials();
  }, []);

  const fetchCredentials = async () => {
    try {
      setLoading(true);
      const data = await credentialAPI.getCredentials();
      setCredentials(data.credentials || []);
    } catch (error: any) {
      message.error('获取登录凭证列表失败');
      setCredentials([]);
    } finally {
      setLoading(false);
    }
  };

  const handleAddCredential = () => {
    setEditingCredential(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEditCredential = (credential: Credential) => {
    setEditingCredential(credential);
    form.setFieldsValue({
      name: credential.name,
      type: credential.type,
      username: credential.username,
    });
    setModalVisible(true);
  };

  const handleDeleteCredential = async (credentialId: number) => {
    try {
      await credentialAPI.deleteCredential(credentialId);
      message.success('登录凭证删除成功');
      fetchCredentials();
    } catch (error: any) {
      message.error('删除登录凭证失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const credentialData: CredentialCreateRequest = {
        ...values,
      };

      if (editingCredential) {
        await credentialAPI.updateCredential(editingCredential.id, credentialData);
        message.success('登录凭证更新成功');
      } else {
        await credentialAPI.createCredential(credentialData);
        message.success('登录凭证添加成功');
      }

      setModalVisible(false);
      fetchCredentials();
    } catch (error: any) {
      if (error.errorFields) {
        // 表单验证错误
        return;
      }
      message.error(editingCredential ? '更新登录凭证失败' : '添加登录凭证失败');
    }
  };

  const columns = [
    {
      title: '凭证名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => (
        <Tag color={type === 'password' ? 'blue' : 'green'} icon={<KeyOutlined />}>
          {type === 'password' ? '密码' : '密钥'}
        </Tag>
      ),
    },
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (createdAt: string) => {
        const date = new Date(createdAt);
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        const seconds = String(date.getSeconds()).padStart(2, '0');
        return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (_text: string, record: Credential) => (
        <Space size="middle">
          <Button icon={<EditOutlined />} onClick={() => handleEditCredential(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这个登录凭证吗？"
            onConfirm={() => handleDeleteCredential(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  if (user?.role !== 'admin') {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Title level={3}>权限不足</Title>
        <p>只有管理员可以管理登录凭证</p>
      </div>
    );
  }

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2} style={{ margin: 0 }}>
          登录凭证管理
        </Title>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={handleAddCredential}
        >
          添加登录凭证
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={credentials || []}
        rowKey="id"
        loading={loading}
        pagination={{
          total: credentials?.length || 0,
          pageSize: 10,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条记录`,
        }}
      />

      <Modal
        title={editingCredential ? '编辑登录凭证' : '添加登录凭证'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            type: 'password',
          }}
        >
          <Form.Item
            name="name"
            label="凭证名称"
            rules={[{ required: true, message: '请输入凭证名称' }]}
          >
            <Input placeholder="例如：生产环境凭证" />
          </Form.Item>

          <Form.Item
            name="type"
            label="认证类型"
            rules={[{ required: true, message: '请选择认证类型' }]}
          >
            <Select>
              <Option value="password">密码认证</Option>
              <Option value="key">密钥认证</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="例如：root, ubuntu" />
          </Form.Item>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) =>
              prevValues.type !== currentValues.type
            }
          >
            {({ getFieldValue }) => {
              const authType = getFieldValue('type');
              return authType === 'password' ? (
                <Form.Item
                  name="password"
                  label="密码"
                  rules={[{ required: true, message: '请输入密码' }]}
                >
                  <Input.Password placeholder="请输入密码" />
                </Form.Item>
              ) : (
                <>
                  <Form.Item
                    name="private_key"
                    label="私钥内容"
                    rules={[{ required: true, message: '请输入私钥内容' }]}
                  >
                    <Input.TextArea
                      rows={4}
                      placeholder="请粘贴私钥内容，例如 -----BEGIN RSA PRIVATE KEY-----"
                    />
                  </Form.Item>
                  <Form.Item
                    name="key_password"
                    label="私钥密码 (可选)"
                  >
                    <Input.Password placeholder="如果私钥有密码保护，请输入密码" />
                  </Form.Item>
                </>
              );
            }}
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Credentials;
