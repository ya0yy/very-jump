import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { message, Spin, Typography } from 'antd';
import { serverAPI } from '../services/api';
import TTYDTerminal from '../components/Terminal/TTYDTerminal';
import type { Server } from '../types';

const Terminal: React.FC = () => {
  const { serverId } = useParams<{ serverId: string }>();
  const [server, setServer] = useState<Server | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchServer = async () => {
      if (!serverId) {
        message.error('服务器ID无效');
        window.close();
        return;
      }

      try {
        const servers = await serverAPI.getServers();
        const targetServer = servers.servers.find(s => s.id === parseInt(serverId));
        
        if (!targetServer) {
          message.error('服务器不存在');
          window.close();
          return;
        }
        
        setServer(targetServer);
      } catch (error) {
        message.error('获取服务器信息失败');
        window.close();
      } finally {
        setLoading(false);
      }
    };

    fetchServer();
  }, [serverId]);

  const handleClose = () => {
    window.close();
  };

  if (loading) {
    return (
      <div style={{ 
        display: 'flex', 
        justifyContent: 'center', 
        alignItems: 'center', 
        height: '100vh',
        flexDirection: 'column'
      }}>
        <Spin size="large" />
        <div style={{ marginTop: 16 }}>
          <Typography.Text>正在连接服务器...</Typography.Text>
        </div>
      </div>
    );
  }

  if (!server) {
    return null;
  }

  return (
    <div style={{ 
      height: '100vh', 
      display: 'flex', 
      flexDirection: 'column',
      background: '#1e1e1e'
    }}>
      <TTYDTerminal
        serverId={server.id}
        serverName={server.name}
        onClose={handleClose}
      />
    </div>
  );
};

export default Terminal;
