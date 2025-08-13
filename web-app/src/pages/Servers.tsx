import React, { useEffect, useState, useCallback } from 'react';
import {
  Table,
  Button,
  Typography,
  Space,
  Modal,
  Form,
  Input,
  Select,
  InputNumber,
  message,
  Popconfirm,
  Tag,
  Tooltip,
  Row,
  Col,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  LinkOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { serverAPI } from '../services/api';
import { useAuthStore } from '../stores/authStore';
import { useAppStore } from '../stores/appStore';

import type { Server, ServerCreateRequest } from '../types';

const { Title } = Typography;
const { Option } = Select;

const Servers: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingServer, setEditingServer] = useState<Server | null>(null);

  const [form] = Form.useForm();
  const { user } = useAuthStore();
  const { servers, setServers, updateServerStatus } = useAppStore();

  const fetchServers = async () => {
    try {
      setLoading(true);
      const data = await serverAPI.getServers();
      // 为每个服务器设置初始状态为检测中
      const serversWithStatus = data.servers.map((server: Server) => ({
        ...server,
        status: 'checking' as const
      }));
      setServers(serversWithStatus);

      // 异步检测每个服务器的状态
      checkServersStatus(serversWithStatus);
    } catch (error: any) {
      message.error('获取服务器列表失败');
    } finally {
      setLoading(false);
    }
  };

  const checkServersStatus = useCallback(async (serverList: Server[]) => {
    // 并发检测所有服务器状态，但逐个更新UI
    const statusPromises = serverList.map(async (server) => {
      try {
        const result = await serverAPI.checkServerStatus(server.id);
        updateServerStatus(server.id, result.status);
      } catch (error) {
        updateServerStatus(server.id, 'unavailable');
      }
    });

    // 等待所有检测完成（虽然UI已经逐个更新了）
    await Promise.all(statusPromises);
  }, [updateServerStatus]);

  useEffect(() => {
    if (!user) return;
    fetchServers();
  }, [user, setServers]);

  const handleAddServer = () => {
    setEditingServer(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEditServer = (server: Server) => {
    setEditingServer(server);
    form.setFieldsValue({
      name: server.name,
      host: server.host,
      port: server.port,
      username: server.username,
      auth_type: server.auth_type,
      description: server.description,
      tags: server.tags || [],
    });
    setModalVisible(true);
  };

  const handleDeleteServer = async (serverId: number) => {
    try {
      await serverAPI.deleteServer(serverId);
      message.success('服务器删除成功');
      fetchServers();
    } catch (error: any) {
      message.error('删除服务器失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const serverData: ServerCreateRequest = {
        ...values,
        port: values.port || 22,
        tags: values.tags || [],
      };

      if (editingServer) {
        await serverAPI.updateServer(editingServer.id, serverData);
        message.success('服务器更新成功');
      } else {
        await serverAPI.createServer(serverData);
        message.success('服务器添加成功');
      }

      setModalVisible(false);
      fetchServers();
    } catch (error: any) {
      if (error.errorFields) {
        // 表单验证错误
        return;
      }
      message.error(editingServer ? '更新服务器失败' : '添加服务器失败');
    }
  };

  const handleConnectServer = (server: Server) => {
    // 在新标签页中打开终端页面
    const terminalUrl = `/terminal/${server.id}`;
    window.open(terminalUrl, '_blank');
  };

  const handleRefreshStatus = async () => {
    if (servers.length === 0) return;

    // 将所有服务器状态设置为检测中
    servers.forEach(server => {
      updateServerStatus(server.id, 'checking');
    });

    // 重新检测状态
    await checkServersStatus(servers);
  };

  const columns = [
    {
      title: '服务器名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '地址',
      dataIndex: 'host',
      key: 'host',
      render: (_text: string, record: Server) => (
        <Typography.Text code>{record.host}:{record.port}</Typography.Text>
      ),
    },
    {
      title: '用户',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: '认证方式',
      dataIndex: 'auth_type',
      key: 'auth_type',
      render: (authType: string) => (
        <Tag color={authType === 'password' ? 'blue' : 'green'}>
          {authType === 'password' ? '密码' : '密钥'}
        </Tag>
      ),
    },
    {
      title: '标签',
      dataIndex: 'tags',
      key: 'tags',
      render: (tags: string[]) => (
        <Space size={[0, 4]} wrap>
          {tags && tags.length > 0 ? (
            tags.map((tag, index) => (
              <Tag key={index} color="default">
                {tag}
              </Tag>
            ))
          ) : (
            <Typography.Text type="secondary">无标签</Typography.Text>
          )}
        </Space>
      ),
    },
    {
      title: '服务器状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const statusConfig = {
          available: { color: 'success', text: '可用' },
          unavailable: { color: 'error', text: '不可用' },
          checking: { color: 'processing', text: '检测中' }
        };
        const config = statusConfig[status as keyof typeof statusConfig] || statusConfig.checking;
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '上次登录时间',
      dataIndex: 'last_login_time',
      key: 'last_login_time',
      render: (lastLoginTime: string) => {
        if (!lastLoginTime) {
          return <Typography.Text type="secondary">从未登录</Typography.Text>;
        }
        const date = new Date(lastLoginTime);
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        const seconds = String(date.getSeconds()).padStart(2, '0');
        const formattedTime = `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
        return <Typography.Text>{formattedTime}</Typography.Text>;
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (_text: string, record: Server) => (
        <Space size="middle">
          <Button type="primary" onClick={() => handleConnectServer(record)} icon={<LinkOutlined />}>
            连接
          </Button>
          {user?.role === 'admin' && (
            <>
              <Tooltip title="编辑">
                <Button icon={<EditOutlined />} onClick={() => handleEditServer(record)} />
              </Tooltip>
              <Popconfirm
                title="确定要删除这个服务器吗？"
                onConfirm={() => handleDeleteServer(record.id)}
                okText="确定"
                cancelText="取消"
              >
                <Tooltip title="删除">
                  <Button danger icon={<DeleteOutlined />} />
                </Tooltip>
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2} style={{ margin: 0 }}>
          服务器管理
        </Title>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={handleRefreshStatus}
          >
            刷新状态
          </Button>
          {user?.role === 'admin' && (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={handleAddServer}
            >
              添加服务器
            </Button>
          )}
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={servers}
        loading={loading}
        rowKey="id"
      />

      <Modal
        title={editingServer ? '编辑服务器' : '添加服务器'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            port: 22,
            auth_type: 'password',
          }}
        >
          <Form.Item
            name="name"
            label="服务器名称"
            rules={[{ required: true, message: '请输入服务器名称' }]}
          >
            <Input placeholder="例如：开发服务器" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={16}>
              <Form.Item
                name="host"
                label="服务器地址"
                rules={[{ required: true, message: '请输入服务器地址' }]}
              >
                <Input placeholder="例如：192.168.1.100 或 server.example.com" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="port"
                label="SSH 端口"
                rules={[{ required: true, message: '请输入SSH端口' }]}
              >
                <InputNumber
                  min={1}
                  max={65535}
                  style={{ width: '100%' }}
                  placeholder="22"
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="username"
                label="SSH 用户名"
                rules={[{ required: true, message: '请输入SSH用户名' }]}
              >
                <Input placeholder="例如：root, ubuntu" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="auth_type"
                label="认证方式"
                rules={[{ required: true, message: '请选择认证方式' }]}
              >
                <Select>
                  <Option value="password">密码认证</Option>
                  <Option value="key">密钥认证</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) =>
              prevValues.auth_type !== currentValues.auth_type
            }
          >
            {({ getFieldValue }) => {
              const authType = getFieldValue('auth_type');
              return authType === 'password' ? (
                <Form.Item
                  name="password"
                  label="SSH 密码"
                  rules={[{ required: true, message: '请输入SSH密码' }]}
                >
                  <Input.Password placeholder="请输入SSH密码" />
                </Form.Item>
              ) : (
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
              );
            }}
          </Form.Item>

          <Form.Item
            name="tags"
            label="标签 (可选)"
            tooltip="用于分类整理服务器，支持多个标签"
          >
            <Select
              mode="tags"
              style={{ width: '100%' }}
              placeholder="输入标签后按回车添加，例如：生产环境、测试服务器"
              tokenSeparators={[',']}
            />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述 (可选)"
          >
            <Input.TextArea
              rows={2}
              placeholder="服务器用途描述"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Servers;