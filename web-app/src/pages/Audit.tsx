import React, { useState, useEffect } from 'react';
import { Card, Table, Tag, Select, Button, Space, Statistic, Row, Col, Alert } from 'antd';
import { 
  WarningOutlined, 
  ClockCircleOutlined,
  UserOutlined,
  DesktopOutlined,
  ReloadOutlined
} from '@ant-design/icons';
import { useAuthStore } from '../stores/authStore';
import api from '../services/api';
import type { ColumnsType } from 'antd/es/table';

const { Option } = Select;

interface AuditLog {
  id: number;
  user_id: number;
  server_id?: number;
  action: string;
  details: string;
  session_id?: string;
  ip_address: string;
  user_agent: string;
  success: boolean;
  error_msg?: string;
  created_at: string;
}

interface SecurityAlert {
  id: number;
  user_id?: number;
  server_id?: number;
  alert_type: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  description: string;
  details: string;
  ip_address: string;
  session_id?: string;
  resolved: boolean;
  resolved_by?: number;
  resolved_at?: string;
  created_at: string;
}

interface AuditStatistics {
  total_sessions: number;
  active_sessions: number;
  total_commands: number;
  failed_logins: number;
  security_alerts: number;
  unresolved_alerts: number;
}

interface TerminalSession {
  id: number;
  session_id: string;
  user_id: number;
  server_id: number;
  start_time: string;
  end_time?: string;
  duration: number;
  command_count: number;
  ip_address: string;
  status: 'active' | 'ended' | 'error';
  created_at: string;
  updated_at: string;
}

const Audit: React.FC = () => {
  const { user } = useAuthStore();
  const [activeTab, setActiveTab] = useState<'logs' | 'alerts' | 'sessions' | 'statistics'>('statistics');
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [securityAlerts, setSecurityAlerts] = useState<SecurityAlert[]>([]);
  const [terminalSessions, setTerminalSessions] = useState<TerminalSession[]>([]);
  const [statistics, setStatistics] = useState<AuditStatistics | null>(null);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState({
    action: '',
    dateRange: null as any,
    resolved: undefined as boolean | undefined,
  });

  // 检查管理员权限
  const isAdmin = user?.role === 'admin';

  useEffect(() => {
    if (activeTab === 'statistics') {
      fetchStatistics();
    } else if (activeTab === 'logs') {
      fetchAuditLogs();
    } else if (activeTab === 'alerts') {
      fetchSecurityAlerts();
    } else if (activeTab === 'sessions') {
      fetchTerminalSessions();
    }
  }, [activeTab, filter]);

  const fetchStatistics = async () => {
    if (!isAdmin) return;
    
    try {
      setLoading(true);
      const response = await api.get('/api/v1/audit/statistics');
      setStatistics(response.data);
    } catch (error) {
      console.error('Failed to fetch statistics:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchAuditLogs = async () => {
    try {
      setLoading(true);
      let url = '/api/v1/audit/logs?page=1&page_size=50';
      if (filter.action) {
        url += `&action=${filter.action}`;
      }
      const response = await api.get(url);
      setAuditLogs(response.data.logs || []);
    } catch (error) {
      console.error('Failed to fetch audit logs:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchSecurityAlerts = async () => {
    if (!isAdmin) return;
    
    try {
      setLoading(true);
      let url = '/api/v1/audit/alerts?page=1&page_size=50';
      if (filter.resolved !== undefined) {
        url += `&resolved=${filter.resolved}`;
      }
      const response = await api.get(url);
      setSecurityAlerts(response.data.alerts || []);
    } catch (error) {
      console.error('Failed to fetch security alerts:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchTerminalSessions = async () => {
    try {
      setLoading(true);
      const response = await api.get('/api/v1/audit/sessions?page=1&page_size=50');
      setTerminalSessions(response.data.sessions || []);
    } catch (error) {
      console.error('Failed to fetch terminal sessions:', error);
    } finally {
      setLoading(false);
    }
  };

  const resolveAlert = async (alertId: number) => {
    try {
      await api.put(`/api/v1/audit/alerts/${alertId}/resolve`);
      fetchSecurityAlerts(); // 刷新列表
    } catch (error) {
      console.error('Failed to resolve alert:', error);
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'red';
      case 'high': return 'orange';
      case 'medium': return 'yellow';
      case 'low': return 'blue';
      default: return 'default';
    }
  };

  const getActionColor = (action: string, success: boolean) => {
    if (!success) return 'red';
    switch (action) {
      case 'terminal_start': return 'green';
      case 'terminal_end': return 'blue';
      case 'login': return 'cyan';
      case 'logout': return 'purple';
      default: return 'default';
    }
  };

  const auditLogColumns: ColumnsType<AuditLog> = [
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (date: string) => new Date(date).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      dataIndex: 'action',
      key: 'action',
      width: 120,
      render: (action: string, record: AuditLog) => (
        <Tag color={getActionColor(action, record.success)}>
          {action}
        </Tag>
      ),
    },
    {
      title: '用户ID',
      dataIndex: 'user_id',
      key: 'user_id',
      width: 80,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip_address',
      key: 'ip_address',
      width: 120,
    },
    {
      title: '状态',
      dataIndex: 'success',
      key: 'success',
      width: 80,
      render: (success: boolean) => (
        <Tag color={success ? 'green' : 'red'}>
          {success ? '成功' : '失败'}
        </Tag>
      ),
    },
    {
      title: '详情',
      dataIndex: 'details',
      key: 'details',
      ellipsis: true,
      render: (details: string) => {
        try {
          const parsed = JSON.parse(details);
          return JSON.stringify(parsed, null, 2);
        } catch {
          return details;
        }
      },
    },
  ];

  const alertColumns: ColumnsType<SecurityAlert> = [
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (date: string) => new Date(date).toLocaleString('zh-CN'),
    },
    {
      title: '严重级别',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: string) => (
        <Tag color={getSeverityColor(severity)}>
          {severity.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: '类型',
      dataIndex: 'alert_type',
      key: 'alert_type',
      width: 150,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip_address',
      key: 'ip_address',
      width: 120,
    },
    {
      title: '状态',
      dataIndex: 'resolved',
      key: 'resolved',
      width: 100,
      render: (resolved: boolean, record: SecurityAlert) => (
        <Space>
          <Tag color={resolved ? 'green' : 'red'}>
            {resolved ? '已解决' : '待处理'}
          </Tag>
          {!resolved && isAdmin && (
            <Button
              size="small"
              type="link"
              onClick={() => resolveAlert(record.id)}
            >
              解决
            </Button>
          )}
        </Space>
      ),
    },
  ];

  const sessionColumns: ColumnsType<TerminalSession> = [
    {
      title: '会话ID',
      dataIndex: 'session_id',
      key: 'session_id',
      width: 200,
      ellipsis: true,
    },
    {
      title: '用户ID',
      dataIndex: 'user_id',
      key: 'user_id',
      width: 80,
    },
    {
      title: '服务器ID',
      dataIndex: 'server_id',
      key: 'server_id',
      width: 80,
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      width: 180,
      render: (date: string) => new Date(date).toLocaleString('zh-CN'),
    },
    {
      title: '持续时间',
      dataIndex: 'duration',
      key: 'duration',
      width: 100,
      render: (duration: number) => `${Math.floor(duration / 60)}分${duration % 60}秒`,
    },
    {
      title: '命令数',
      dataIndex: 'command_count',
      key: 'command_count',
      width: 80,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : status === 'ended' ? 'blue' : 'red'}>
          {status === 'active' ? '活跃' : status === 'ended' ? '已结束' : '错误'}
        </Tag>
      ),
    },
    {
      title: 'IP地址',
      dataIndex: 'ip_address',
      key: 'ip_address',
      width: 120,
    },
  ];

  if (!isAdmin && activeTab !== 'logs' && activeTab !== 'sessions') {
    return (
      <Card>
        <Alert
          message="权限不足"
          description="您需要管理员权限才能查看此页面"
          type="warning"
          showIcon
        />
      </Card>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      <Card 
        title="审计管理" 
        extra={
          <Space>
            <Button 
              icon={<ReloadOutlined />} 
              onClick={() => {
                if (activeTab === 'statistics') fetchStatistics();
                else if (activeTab === 'logs') fetchAuditLogs();
                else if (activeTab === 'alerts') fetchSecurityAlerts();
                else if (activeTab === 'sessions') fetchTerminalSessions();
              }}
            >
              刷新
            </Button>
          </Space>
        }
        tabList={[
          { key: 'statistics', tab: '统计概览' },
          { key: 'logs', tab: '审计日志' },
          { key: 'alerts', tab: '安全告警' },
          { key: 'sessions', tab: '终端会话' },
        ]}
        activeTabKey={activeTab}
        onTabChange={(key) => setActiveTab(key as any)}
      >
        {activeTab === 'statistics' && statistics && (
          <Row gutter={16}>
            <Col span={6}>
              <Card>
                <Statistic
                  title="总会话数"
                  value={statistics.total_sessions}
                  prefix={<DesktopOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="活跃会话"
                  value={statistics.active_sessions}
                  prefix={<ClockCircleOutlined />}
                  valueStyle={{ color: '#3f8600' }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="失败登录（24h）"
                  value={statistics.failed_logins}
                  prefix={<UserOutlined />}
                  valueStyle={{ color: statistics.failed_logins > 0 ? '#cf1322' : '#3f8600' }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="未解决告警"
                  value={statistics.unresolved_alerts}
                  prefix={<WarningOutlined />}
                  valueStyle={{ color: statistics.unresolved_alerts > 0 ? '#cf1322' : '#3f8600' }}
                />
              </Card>
            </Col>
          </Row>
        )}

        {activeTab === 'logs' && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Space>
                <Select
                  placeholder="筛选操作类型"
                  allowClear
                  style={{ width: 200 }}
                  onChange={(value) => setFilter({ ...filter, action: value || '' })}
                >
                  <Option value="terminal_start">终端启动</Option>
                  <Option value="terminal_end">终端结束</Option>
                  <Option value="login">登录</Option>
                  <Option value="logout">登出</Option>
                </Select>
              </Space>
            </div>
            <Table
              columns={auditLogColumns}
              dataSource={auditLogs}
              rowKey="id"
              loading={loading}
              pagination={{ pageSize: 20 }}
              scroll={{ x: true }}
            />
          </div>
        )}

        {activeTab === 'alerts' && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Space>
                <Select
                  placeholder="筛选解决状态"
                  allowClear
                  style={{ width: 200 }}
                  onChange={(value) => setFilter({ ...filter, resolved: value })}
                >
                  <Option value={false}>待处理</Option>
                  <Option value={true}>已解决</Option>
                </Select>
              </Space>
            </div>
            <Table
              columns={alertColumns}
              dataSource={securityAlerts}
              rowKey="id"
              loading={loading}
              pagination={{ pageSize: 20 }}
              scroll={{ x: true }}
            />
          </div>
        )}

        {activeTab === 'sessions' && (
          <Table
            columns={sessionColumns}
            dataSource={terminalSessions}
            rowKey="id"
            loading={loading}
            pagination={{ pageSize: 20 }}
            scroll={{ x: true }}
          />
        )}
      </Card>
    </div>
  );
};

export default Audit;
