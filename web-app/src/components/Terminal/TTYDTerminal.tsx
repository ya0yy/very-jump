import React, { useEffect, useState, useRef } from 'react';
import { Card, Button, Space, Typography, message, Spin } from 'antd';
import { CloseOutlined, ReloadOutlined } from '@ant-design/icons';
import type { TerminalProps } from '../../types';
import api from '../../services/api';
import { useAuthStore } from '../../stores/authStore';

const { Text } = Typography;

interface TTYDSession {
  session_id: string;
  port: number;
  url: string;
}

const TTYDTerminal: React.FC<TerminalProps> = ({ serverId, serverName, onClose }) => {
  const [session, setSession] = useState<TTYDSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { isAuthenticated, token, checkAuth } = useAuthStore();
  const iframeRef = useRef<HTMLIFrameElement>(null);

  const startTerminalSession = async () => {
    try {
      setLoading(true);
      setError(null);

      if (!isAuthenticated || !token) {
        throw new Error('用户未认证，请重新登录');
      }

      const response = await api.post(`/terminal/start/${serverId}`);
      const sessionData: TTYDSession = response.data;

      if (!sessionData.url) {
        throw new Error('服务器返回的URL为空');
      }

      const absoluteUrl = new URL(sessionData.url, window.location.origin);
      absoluteUrl.searchParams.append('token', token!);

      console.log('Final URL with token:', absoluteUrl.href);

      setSession({
        ...sessionData,
        url: absoluteUrl.href
      });

      message.success('终端会话已启动');
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || err.message || '启动终端失败';
      setError(errorMsg);
      message.error(errorMsg);

      if (err.response?.status === 401) {
        checkAuth();
      }
    } finally {
      setLoading(false);
    }
  };

  const stopTerminalSession = async (sessionId: string) => {
    try {
      await api.post(`/terminal/stop/${sessionId}`);
    } catch (err: any) {
      console.error('停止终端会话失败:', err);
    }
  };

  useEffect(() => {
    if (!isAuthenticated || !token) {
      checkAuth();
    }
    
    const timer = setTimeout(() => {
      if (isAuthenticated && token) {
        startTerminalSession();
      } else {
        setError('认证状态无效，请重新登录');
        setLoading(false);
      }
    }, 100);

    return () => {
      clearTimeout(timer);
      if (session) {
        stopTerminalSession(session.session_id);
      }
    };
  }, [serverId, isAuthenticated, token]);

  useEffect(() => {
    if (iframeRef.current && session?.url) {
      console.log("Programmatically setting iframe src to:", session.url);
      iframeRef.current.src = session.url;
    }
  }, [session?.url]);

  const handleClose = async () => {
    if (session) {
      await stopTerminalSession(session.session_id);
    }
    onClose();
  };

  if (loading) {
    return (
      <Card
        title={
          <Space>
            <Text strong>{serverName}</Text>
            <Text type="secondary">启动中...</Text>
          </Space>
        }
        style={{ height: '100vh', margin: 0 }}
        bodyStyle={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: 'calc(100vh - 80px)',
        }}
      >
        <Spin size="large" tip="正在启动终端会话..." />
      </Card>
    );
  }

  if (error || !session) {
    return (
      <Card
        title={
          <Space>
            <Text strong>{serverName}</Text>
            <Text type="danger">启动失败</Text>
          </Space>
        }
        extra={
          <Space>
            <Button
              type="primary"
              icon={<ReloadOutlined />}
              onClick={startTerminalSession}
            >
              重试
            </Button>
            <Button
              type="text"
              danger
              icon={<CloseOutlined />}
              onClick={handleClose}
            />
          </Space>
        }
        style={{ height: '100vh', margin: 0 }}
        bodyStyle={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: 'calc(100vh - 80px)',
        }}
      >
        <div style={{ textAlign: 'center' }}>
          <Text type="danger">{error}</Text>
          <br />
          <Button type="primary" onClick={startTerminalSession} style={{ marginTop: 16 }}>
            重新启动
          </Button>
        </div>
      </Card>
    );
  }

  return (
    <Card
      title={
        <Space>
          <Text strong>{serverName}</Text>
          <Text type="secondary">TTYD终端</Text>
          <Text
            type="success"
            style={{ fontSize: '12px' }}
          >
            ● 已连接
          </Text>
        </Space>
      }
      extra={
        <Space>
          <Button
            type="text"
            icon={<ReloadOutlined />}
            onClick={startTerminalSession}
            title="重新连接"
          />
          <Button
            type="text"
            danger
            icon={<CloseOutlined />}
            onClick={handleClose}
          />
        </Space>
      }
      bodyStyle={{
        padding: 0,
        height: 'calc(100vh - 80px)',
        overflow: 'hidden',
      }}
      style={{
        height: '100vh',
        margin: 0,
      }}
    >
      <iframe
        ref={iframeRef}
        style={{
          width: '100%',
          height: '100%',
          border: 'none',
          backgroundColor: '#000',
        }}
        title={`Terminal for ${serverName}`}
      />
    </Card>
  );
};

export default TTYDTerminal;