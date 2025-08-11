import React, { useEffect } from 'react';
import { Layout, Menu, Avatar, Dropdown, Typography, Space, Badge } from 'antd';
import {
  DatabaseOutlined,
  HistoryOutlined,
  AuditOutlined,
  SafetyOutlined,
  UserOutlined,
  LogoutOutlined,
  DashboardOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import { useAuthStore } from '../../stores/authStore';
import { useAppStore } from '../../stores/appStore';
import { sessionAPI } from '../../services/api';

const { Header, Sider, Content } = Layout;
const { Title } = Typography;

const AppLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { activeSessions, setActiveSessions } = useAppStore();

  // 获取活跃会话数
  useEffect(() => {
    const fetchActiveSessions = async () => {
      const token = localStorage.getItem('token');
      if (!token) return;
      try {
        const data = await sessionAPI.getActiveSessions();
        setActiveSessions(data.active_sessions);
      } catch (error) {
        console.error('Failed to fetch active sessions:', error);
      }
    };

    fetchActiveSessions();
    
    // 每30秒更新一次
    const interval = setInterval(fetchActiveSessions, 30000);
    return () => clearInterval(interval);
  }, [setActiveSessions]);

  const menuItems = [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: '仪表板',
    },
    {
      key: '/servers',
      icon: <DatabaseOutlined />,
      label: '服务器管理',
    },
    {
      key: '/sessions',
      icon: <HistoryOutlined />,
      label: '会话历史',
    },
    {
      key: '/audit-logs',
      icon: <AuditOutlined />,
      label: '审计日志',
    },
    {
      key: '/audit',
      icon: <SafetyOutlined />,
      label: '安全审计',
    },
    ...(user?.role === 'admin' ? [{
      key: '/users',
      icon: <UserOutlined />,
      label: '用户管理',
    }] : []),
  ];

  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人信息',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      danger: true,
    },
  ];

  const handleMenuClick = ({ key }: { key: string }) => {
    navigate(key);
  };

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') {
      logout();
      navigate('/login');
    } else if (key === 'profile') {
      // TODO: 打开个人信息页面
    }
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ 
        background: '#001529', 
        padding: '0 24px',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center'
      }}>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <Title level={3} style={{ color: 'white', margin: 0 }}>
            🚀 Very Jump
          </Title>
          <span style={{ color: '#999', marginLeft: 16 }}>
            轻量级跳板机管理平台
          </span>
        </div>
        
        <Space>
          <Badge count={activeSessions} showZero>
            <span style={{ color: '#999' }}>活跃会话</span>
          </Badge>
          
          <Dropdown
            menu={{
              items: userMenuItems,
              onClick: handleUserMenuClick,
            }}
            placement="bottomRight"
          >
            <Space style={{ cursor: 'pointer', color: 'white' }}>
              <Avatar size="small" icon={<UserOutlined />} />
              <span>{user?.username}</span>
              <span style={{ fontSize: '12px', color: '#999' }}>
                ({user?.role === 'admin' ? '管理员' : '用户'})
              </span>
            </Space>
          </Dropdown>
        </Space>
      </Header>

      <Layout>
        <Sider
          width={250}
          style={{
            background: '#fff',
            boxShadow: '2px 0 8px rgba(0,0,0,0.05)',
          }}
        >
          <Menu
            mode="inline"
            selectedKeys={[location.pathname]}
            items={menuItems}
            onClick={handleMenuClick}
            style={{ borderRight: 0, paddingTop: 16 }}
          />
        </Sider>

        <Content
          style={{
            padding: '24px',
            background: '#f5f5f5',
            overflow: 'auto',
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

export default AppLayout;
