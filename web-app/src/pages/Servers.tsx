import React, { useEffect, useState } from 'react';
import {
  Card,
  Row,
  Col,
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
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  LinkOutlined,
  DatabaseOutlined,
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
  const { servers, setServers } = useAppStore();

  const fetchServers = async () => {
    try {
      setLoading(true);
      const data = await serverAPI.getServers();
      setServers(data.servers);
    } catch (error: any) {
      message.error('获取服务器列表失败');
    } finally {
      setLoading(false);
    }
  };

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
    // 在新窗口中打开终端页面
    const terminalUrl = `/terminal/${server.id}`;
    window.open(terminalUrl, '_blank', 'width=1200,height=800,scrollbars=yes,resizable=yes');
  };



  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2} style={{ margin: 0 }}>
          服务器管理
        </Title>
        {user?.role === 'admin' && (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={handleAddServer}
          >
            添加服务器
          </Button>
        )}
      </div>

      <Row gutter={[16, 16]}>
        {servers.map((server) => (
          <Col xs={24} sm={12} lg={8} xl={6} key={server.id}>
            <Card
              title={
                <Space>
                  <DatabaseOutlined />
                  <span>{server.name}</span>
                </Space>
              }
              extra={
                user?.role === 'admin' && (
                  <Space>
                    <Tooltip title="编辑">
                      <Button
                        type="text"
                        size="small"
                        icon={<EditOutlined />}
                        onClick={() => handleEditServer(server)}
                      />
                    </Tooltip>
                    <Popconfirm
                      title="确定要删除这个服务器吗？"
                      onConfirm={() => handleDeleteServer(server.id)}
                      okText="确定"
                      cancelText="取消"
                    >
                      <Tooltip title="删除">
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                        />
                      </Tooltip>
                    </Popconfirm>
                  </Space>
                )
              }
              actions={[
                <Button
                  key="connect"
                  type="primary"
                  icon={<LinkOutlined />}
                  onClick={() => handleConnectServer(server)}
                >
                  连接
                </Button>
              ]}
            >
              <Space direction="vertical" style={{ width: '100%' }}>
                <div>
                  <Typography.Text strong>地址：</Typography.Text>
                  <Typography.Text code>{server.host}:{server.port}</Typography.Text>
                </div>
                <div>
                  <Typography.Text strong>用户：</Typography.Text>
                  <Typography.Text>{server.username}</Typography.Text>
                </div>
                <div>
                  <Typography.Text strong>认证：</Typography.Text>
                  <Tag color={server.auth_type === 'password' ? 'blue' : 'green'}>
                    {server.auth_type === 'password' ? '密码' : '密钥'}
                  </Tag>
                </div>
                {server.description && (
                  <div>
                    <Typography.Text strong>描述：</Typography.Text>
                    <Typography.Text type="secondary">{server.description}</Typography.Text>
                  </div>
                )}
              </Space>
            </Card>
          </Col>
        ))}

        {servers.length === 0 && !loading && (
          <Col span={24}>
            <Card style={{ textAlign: 'center', padding: '40px 0' }}>
              <Space direction="vertical">
                <DatabaseOutlined style={{ fontSize: 48, color: '#d9d9d9' }} />
                <Typography.Text type="secondary">
                  暂无服务器
                </Typography.Text>
                {user?.role === 'admin' && (
                  <Button type="primary" onClick={handleAddServer}>
                    添加第一台服务器
                  </Button>
                )}
              </Space>
            </Card>
          </Col>
        )}
      </Row>

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
