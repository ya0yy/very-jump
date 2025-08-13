import React from 'react';
import { Card, Space, Tag, Typography } from 'antd';
import {
  DatabaseOutlined,
} from '@ant-design/icons';
import type { Server } from '../types';

interface ServerCardProps {
  server: Server;
  onConnect: (server: Server) => void;
}

const ServerCard: React.FC<ServerCardProps> = ({ server, onConnect }) => {
  return (
    <Card
      hoverable
      onClick={() => onConnect(server)}
      title={
        <Space>
          <DatabaseOutlined />
          <span>{server.name}</span>
        </Space>
      }
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
  );
};

export default ServerCard;