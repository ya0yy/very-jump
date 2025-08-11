import React, { useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, theme } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { useAuthStore } from './stores/authStore';
import AppLayout from './components/Layout/AppLayout';
import LoginForm from './components/Auth/LoginForm';
import Dashboard from './pages/Dashboard';
import Servers from './pages/Servers';
import Sessions from './pages/Sessions';
import AuditLogs from './pages/AuditLogs';
import Audit from './pages/Audit';
import Users from './pages/Users';
import Terminal from './pages/Terminal';
import './App.css';

// 受保护的路由组件
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();
  const tokenInStorage = typeof window !== 'undefined' ? localStorage.getItem('token') : null;

  if (!isAuthenticated || !tokenInStorage) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

// 管理员路由组件
const AdminRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user } = useAuthStore();
  const tokenInStorage = typeof window !== 'undefined' ? localStorage.getItem('token') : null;

  if (!tokenInStorage || user?.role !== 'admin') {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
};

const App: React.FC = () => {
  const { isAuthenticated, checkAuth } = useAuthStore();
  const [isInitialized, setIsInitialized] = React.useState(false);

  // 检查本地存储的认证状态
  useEffect(() => {
    checkAuth();
    setIsInitialized(true);
  }, [checkAuth]);

  // 在初始化完成前显示加载状态
  if (!isInitialized) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
      Loading...
    </div>;
  }

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#667eea',
          borderRadius: 8,
        },
        algorithm: theme.defaultAlgorithm,
      }}
    >
      <Router>
        <Routes>
          {/* 登录页面 */}
          <Route
            path="/login"
            element={
              isAuthenticated ? <Navigate to="/dashboard" replace /> : <LoginForm />
            }
          />
          
          {/* 终端页面 - 独立页面，不需要AppLayout包装 */}
          <Route 
            path="/terminal/:serverId" 
            element={
              <ProtectedRoute>
                <Terminal />
              </ProtectedRoute>
            } 
          />
          
          {/* 受保护的路由 */}
          <Route path="/" element={<ProtectedRoute><AppLayout /></ProtectedRoute>}>
            <Route index element={<Navigate to="/dashboard" replace />} />
            <Route path="dashboard" element={<Dashboard />} />
            <Route path="servers" element={<Servers />} />
            <Route path="sessions" element={<Sessions />} />
            <Route path="audit-logs" element={<AuditLogs />} />
            <Route path="audit" element={<Audit />} />
            
            {/* 管理员路由 */}
            <Route
              path="users"
              element={
                <AdminRoute>
                  <Users />
                </AdminRoute>
              }
            />
          </Route>
        </Routes>
      </Router>
    </ConfigProvider>
  );
};

export default App;