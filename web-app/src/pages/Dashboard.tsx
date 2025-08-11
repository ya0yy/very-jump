import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Typography, Space, Alert } from 'antd';
import {
  DatabaseOutlined,
  HistoryOutlined,
  UserOutlined,
  LinkOutlined,
} from '@ant-design/icons';
import { sessionAPI, serverAPI } from '../services/api';
import { useAuthStore } from '../stores/authStore';

const { Title } = Typography;

interface DashboardStats {
  total_servers: number;
  active_sessions: number;
  total_users: number;
  total_sessions: number;
}

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { user } = useAuthStore();

  useEffect(() => {
    const fetchDashboardData = async () => {
      // 确保用户已认证再加载数据
      if (!user) {
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        
        // 获取基础统计数据
        const [serversData, activeSessionsData] = await Promise.all([
          serverAPI.getServers(),
          sessionAPI.getActiveSessions(),
        ]);

        const dashboardStats: DashboardStats = {
          total_servers: serversData.servers.length,
          active_sessions: activeSessionsData.active_sessions,
          total_users: 0, // 将从用户API获取
          total_sessions: 0, // 将从会话API获取
        };

        setStats(dashboardStats);
      } catch (err: any) {
        setError('加载仪表板数据失败');
        console.error('Dashboard data fetch error:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchDashboardData();
  }, [user]); // 依赖user变化

  if (error) {
    return (
      <div>
        <Title level={2}>仪表板</Title>
        <Alert
          message="加载失败"
          description={error}
          type="error"
          showIcon
        />
      </div>
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Space direction="vertical">
          <Title level={2} style={{ margin: 0 }}>
            仪表板
          </Title>
          <Typography.Text type="secondary">
            欢迎回来，{user?.username}！
          </Typography.Text>
        </Space>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card loading={loading}>
            <Statistic
              title="服务器总数"
              value={stats?.total_servers || 0}
              prefix={<DatabaseOutlined />}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card loading={loading}>
            <Statistic
              title="活跃会话"
              value={stats?.active_sessions || 0}
              prefix={<LinkOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card loading={loading}>
            <Statistic
              title="历史会话"
              value={stats?.total_sessions || 0}
              prefix={<HistoryOutlined />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card loading={loading}>
            <Statistic
              title="用户总数"
              value={stats?.total_users || 0}
              prefix={<UserOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} lg={12}>
          <Card title="系统状态" loading={loading}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span>服务状态</span>
                <span style={{ color: '#52c41a' }}>● 运行正常</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span>数据库状态</span>
                <span style={{ color: '#52c41a' }}>● 连接正常</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span>WebSocket</span>
                <span style={{ color: '#52c41a' }}>● 可用</span>
              </div>
            </Space>
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="快速操作" loading={loading}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Typography.Link href="/servers">
                📖 管理服务器
              </Typography.Link>
              <Typography.Link href="/sessions">
                📊 查看会话历史
              </Typography.Link>
              {user?.role === 'admin' && (
                <>
                  <Typography.Link href="/users">
                    👥 用户管理
                  </Typography.Link>
                  <Typography.Link href="/audit">
                    🔍 审计日志
                  </Typography.Link>
                </>
              )}
            </Space>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
