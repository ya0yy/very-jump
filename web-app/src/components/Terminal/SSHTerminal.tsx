import React, { useEffect, useRef, useState } from 'react';
import { Card, Button, Space, Typography, message } from 'antd';
import { CloseOutlined } from '@ant-design/icons';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import type { TerminalProps, WebSocketMessage } from '../../types';

const { Text } = Typography;

const SSHTerminal: React.FC<TerminalProps> = ({ serverId, serverName, onClose }) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminal = useRef<Terminal | null>(null);
  const fitAddon = useRef<FitAddon | null>(null);
  const websocket = useRef<WebSocket | null>(null);
  const [, setIsConnected] = useState(false);

  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('connecting');

  useEffect(() => {
    if (!terminalRef.current) return;

    // 初始化终端
    terminal.current = new Terminal({
      fontFamily: '"Cascadia Code", "Fira Code", "Monaco", "Menlo", "Ubuntu Mono", monospace',
      fontSize: 14,
      lineHeight: 1.2,
      cursorBlink: true,
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#d4d4d4',
        selectionBackground: '#264f78',
        black: '#000000',
        red: '#cd3131',
        green: '#0dbc79',
        yellow: '#e5e510',
        blue: '#2472c8',
        magenta: '#bc3fbc',
        cyan: '#11a8cd',
        white: '#e5e5e5',
        brightBlack: '#666666',
        brightRed: '#f14c4c',
        brightGreen: '#23d18b',
        brightYellow: '#f5f543',
        brightBlue: '#3b8eea',
        brightMagenta: '#d670d6',
        brightCyan: '#29b8db',
        brightWhite: '#e5e5e5',
      },
    });

    fitAddon.current = new FitAddon();
    terminal.current.loadAddon(fitAddon.current);
    terminal.current.open(terminalRef.current);
    fitAddon.current.fit();

    // 连接WebSocket
    connectWebSocket();

    // 监听终端输入
    terminal.current.onData((data) => {
      if (websocket.current && websocket.current.readyState === WebSocket.OPEN) {
        const message: WebSocketMessage = {
          type: 'input',
          data: data,
        };
        websocket.current.send(JSON.stringify(message));
      }
    });

    // 监听窗口大小变化
    const handleResize = () => {
      if (fitAddon.current && terminal.current) {
        fitAddon.current.fit();
        
        // 发送终端大小变化消息
        if (websocket.current && websocket.current.readyState === WebSocket.OPEN) {
          const cols = terminal.current.cols;
          const rows = terminal.current.rows;
          const message: WebSocketMessage = {
            type: 'resize',
            data: { cols, rows },
          };
          websocket.current.send(JSON.stringify(message));
        }
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      if (websocket.current) {
        websocket.current.close();
      }
      if (terminal.current) {
        terminal.current.dispose();
      }
    };
  }, [serverId]);

  const connectWebSocket = () => {
    const token = localStorage.getItem('token');
    if (!token) {
      message.error('未找到认证令牌');
      return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/ssh/${serverId}?token=${encodeURIComponent(token)}`;

    websocket.current = new WebSocket(wsUrl);

    websocket.current.onopen = () => {
      setIsConnected(true);
      setConnectionStatus('connected');

      if (terminal.current) {
        terminal.current.writeln(`\x1b[32m正在连接到服务器 ${serverName}...\x1b[0m`);
      }
    };

    websocket.current.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);
        
        if (terminal.current && (message.type === 'output' || message.type === 'stdout' || message.type === 'stderr')) {
          terminal.current.write(message.data as string);
        }
      } catch (error) {
        // 处理非JSON消息
        if (terminal.current) {
          terminal.current.write(event.data);
        }
      }
    };

    websocket.current.onclose = () => {
      setIsConnected(false);
      setConnectionStatus('disconnected');
      
      if (terminal.current) {
        terminal.current.writeln('\x1b[31m\r\n连接已断开\x1b[0m');
      }
    };

    websocket.current.onerror = (error) => {
      setConnectionStatus('error');
      console.error('WebSocket error:', error);
      
      if (terminal.current) {
        terminal.current.writeln('\x1b[31m\r\n连接错误\x1b[0m');
      }
      
      message.error('SSH连接失败');
    };
  };

  const handleClose = () => {
    if (websocket.current) {
      websocket.current.send(JSON.stringify({
        type: 'close',
        data: '',
      }));
      websocket.current.close();
    }
    onClose();
  };



  const terminalContent = (
    <Card
      title={
        <Space>
          <Text strong>{serverName}</Text>
          <Text type="secondary">SSH 终端</Text>
          <Text 
            type={connectionStatus === 'connected' ? 'success' : 'danger'}
            style={{ fontSize: '12px' }}
          >
            ● {connectionStatus === 'connected' ? '已连接' : 
                connectionStatus === 'connecting' ? '连接中...' : 
                connectionStatus === 'error' ? '连接错误' : '已断开'}
          </Text>
        </Space>
      }
      extra={
        <Button
          type="text"
          danger
          icon={<CloseOutlined />}
          onClick={handleClose}
        />
      }
      bodyStyle={{ 
        padding: 0, 
        background: '#1e1e1e',
        height: 'calc(100vh - 80px)', // 留出标题栏高度
      }}
      style={{
        height: '100vh',
        margin: 0,
      }}
    >
      <div
        ref={terminalRef}
        style={{
          height: '100%',
          padding: '16px',
        }}
      />
    </Card>
  );

  return terminalContent;
};

export default SSHTerminal;
