import React, { useEffect, useState } from 'react';
import {
  Table,
  Typography,
  Space,
  Tag,
  Button,
  Popconfirm,
  message,
  Card,
} from 'antd';
import {
  StopOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { sessionAPI } from '../services/api';
import { useAppStore } from '../stores/appStore';
import SessionReplay from '../components/SessionReplay';
import type { Session } from '../types';

const { Title } = Typography;

const Sessions: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [replayVisible, setReplayVisible] = useState(false);
  const [selectedSessionId, setSelectedSessionId] = useState<string>('');
  const { sessions, setSessions } = useAppStore();

  const fetchSessions = async () => {
    try {
      setLoading(true);
      const data = await sessionAPI.getSessions();
      setSessions(data.sessions);
    } catch (error: any) {
      message.error('获取会话列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSessions();
  }, [setSessions]);

  const handleCloseSession = async (sessionId: string) => {
    try {
      await sessionAPI.closeSession(sessionId);
      message.success('会话已关闭');
      fetchSessions();
    } catch (error: any) {
      message.error('关闭会话失败');
    }
  };

  const handleViewSession = (sessionId: string) => {
    setSelectedSessionId(sessionId);
    setReplayVisible(true);
  };

  const handleCloseReplay = () => {
    setReplayVisible(false);
    setSelectedSessionId('');
  };

  const columns: ColumnsType<Session> = [
    {
      title: '会话ID',
      dataIndex: 'id',
      key: 'id',
      render: (id: string) => (
        <Typography.Text code>{id.slice(0, 8)}...</Typography.Text>
      ),
    },
    {
      title: '服务器',
      dataIndex: 'server_name',
      key: 'server_name',
      render: (name: string) => name || '未知服务器',
    },
    {
      title: '用户',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const statusConfig = {
          active: { color: 'green', text: '活跃' },
          closed: { color: 'default', text: '已关闭' },
          error: { color: 'red', text: '错误' },
        };
        const config = statusConfig[status as keyof typeof statusConfig] || 
                      { color: 'default', text: status };
        
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '客户端IP',
      dataIndex: 'client_ip',
      key: 'client_ip',
      render: (ip: string) => <Typography.Text code>{ip}</Typography.Text>,
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      render: (time: string) => new Date(time).toLocaleString(),
    },
    {
      title: '结束时间',
      dataIndex: 'end_time',
      key: 'end_time',
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, session: Session) => (
        <Space>
          {session.status === 'active' ? (
            <Popconfirm
              title="确定要关闭这个会话吗？"
              onConfirm={() => handleCloseSession(session.id)}
              okText="确定"
              cancelText="取消"
            >
              <Button
                type="primary"
                danger
                size="small"
                icon={<StopOutlined />}
              >
                关闭
              </Button>
            </Popconfirm>
          ) : (
            <Button
              type="default"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => handleViewSession(session.id)}
            >
              回放
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Title level={2} style={{ margin: 0 }}>
          会话历史
        </Title>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={sessions}
          loading={loading}
          rowKey="id"
          pagination={{
            pageSize: 20,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
          }}
          scroll={{ x: 1000 }}
        />
      </Card>

      <SessionReplay
        sessionId={selectedSessionId}
        visible={replayVisible}
        onClose={handleCloseReplay}
      />
    </div>
  );
};

export default Sessions;
