import React, { useEffect, useRef, useState } from 'react';
import { Modal, message, Button, Space, Slider, Typography, Card } from 'antd';
import { PlayCircleOutlined, PauseCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import { sessionAPI } from '../services/api';
import type { Session } from '../types';
import Convert from 'ansi-to-html';

const { Text } = Typography;

interface SessionReplayProps {
  sessionId: string;
  visible: boolean;
  onClose: () => void;
}

interface AsciinemaRecord {
  version: number;
  width: number;
  height: number;
  timestamp: number;
  command?: string;
  title?: string;
  env?: Record<string, string>;
}

interface AsciinemaEvent {
  time: number;
  type: string;
  data: string;
}

const SessionReplay: React.FC<SessionReplayProps> = ({ sessionId, visible, onClose }) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const [loading, setLoading] = useState(false);
  const [session, setSession] = useState<Session | null>(null);
  const [hasRecording, setHasRecording] = useState(false);
  const [recordingData, setRecordingData] = useState<AsciinemaEvent[] | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const playStartTimeRef = useRef<number>(0);
  const pausedAtRef = useRef<number>(0);
  const convertRef = useRef<Convert | null>(null);

  useEffect(() => {
    // 初始化ANSI转换器
    if (!convertRef.current) {
      convertRef.current = new Convert({
        fg: '#FFF',
        bg: '#000',
        newline: false,
        escapeXML: false,
        stream: false
      });
    }
    
    if (visible && sessionId) {
      fetchReplayInfo();
    }
  }, [visible, sessionId]);

  useEffect(() => {
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  const fetchReplayInfo = async () => {
    try {
      setLoading(true);
      const data = await sessionAPI.getReplayInfo(sessionId);
      setSession(data.session);
      setHasRecording(data.has_recording);
      
      if (data.has_recording) {
        await fetchRecordingData();
      }
    } catch (error: any) {
      message.error('获取会话回放信息失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchRecordingData = async () => {
    try {
      const blob = await sessionAPI.getRecordingFile(sessionId);
      const text = await blob.text();
      
      // 解析 asciinema 格式
      const lines = text.trim().split('\n');
      JSON.parse(lines[0]) as AsciinemaRecord; // header info
      const events: AsciinemaEvent[] = [];
      
      for (let i = 1; i < lines.length; i++) {
        if (lines[i].trim()) {
          try {
            const event = JSON.parse(lines[i]);
            events.push({
              time: event[0],
              type: event[1],
              data: event[2]
            });
          } catch (e) {
            console.warn('Failed to parse event:', lines[i]);
          }
        }
      }
      
      setRecordingData(events);
      setDuration(events.length > 0 ? events[events.length - 1].time : 0);
      
    } catch (error: any) {
      message.error('获取录制文件失败');
    }
  };

  const renderTerminalOutput = (upToTime: number) => {
    if (!recordingData || !terminalRef.current || !convertRef.current) return;
    
    let output = '';
    for (const event of recordingData) {
      if (event.time <= upToTime && event.type === 'o') {
        let eventData = event.data;
        
        // 处理ttyd消息格式，剥离消息类型前缀（兼容旧录制文件）
        if (eventData.length > 0) {
          const firstChar = eventData.charAt(0);
          if (firstChar === '0') {
            // '0' = OUTPUT，去除前缀
            eventData = eventData.substring(1);
          } else if (firstChar === '1' || firstChar === '2') {
            // '1' = SET_WINDOW_TITLE, '2' = SET_PREFERENCES，忽略这些消息
            continue;
          }
        }
        
        output += eventData;
      }
    }
    
    // 清理OSC (Operating System Command) 转义序列，特别是窗口标题设置
    // 格式: ESC ] 0 ; title BEL 或 ESC ] 0 ; title ESC \
    output = output.replace(/\x1b\]0;[^\x07\x1b]*[\x07\x1b\\]/g, '');
    // 还要处理其他OSC序列: ESC ] number ; data BEL/ST
    output = output.replace(/\x1b\]\d+;[^\x07\x1b]*[\x07\x1b\\]/g, '');
    
    // 使用ansi-to-html处理ANSI转义序列
    const htmlOutput = convertRef.current.toHtml(output);
    terminalRef.current.innerHTML = `<pre style="margin: 0; white-space: pre-wrap; font-family: monospace; font-size: 14px; line-height: 1.4;">${htmlOutput}</pre>`;
  };

  const play = () => {
    if (!recordingData) return;
    
    setIsPlaying(true);
    playStartTimeRef.current = Date.now() - (pausedAtRef.current * 1000);
    
    intervalRef.current = setInterval(() => {
      const elapsed = (Date.now() - playStartTimeRef.current) / 1000 * playbackSpeed;
      const newTime = Math.min(elapsed, duration);
      setCurrentTime(newTime);
      renderTerminalOutput(newTime);
      
      if (newTime >= duration) {
        pause();
      }
    }, 50);
  };

  const pause = () => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    setIsPlaying(false);
    pausedAtRef.current = currentTime;
  };

  const restart = () => {
    pause();
    setCurrentTime(0);
    pausedAtRef.current = 0;
    renderTerminalOutput(0);
  };

  const seekTo = (time: number) => {
    const wasPlaying = isPlaying;
    pause();
    setCurrentTime(time);
    pausedAtRef.current = time;
    renderTerminalOutput(time);
    
    if (wasPlaying && time < duration) {
      setTimeout(play, 100);
    }
  };

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  };

  return (
    <Modal
      title={`会话回放 - ${session?.server_name || '未知服务器'}`}
      open={visible}
      onCancel={onClose}
      footer={null}
      width="90%"
      style={{ top: 20 }}
      bodyStyle={{ padding: '20px' }}
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: '50px' }}>
          加载中...
        </div>
      ) : !hasRecording ? (
        <div style={{ textAlign: 'center', padding: '50px' }}>
          <Text type="secondary">该会话没有录制文件</Text>
        </div>
      ) : (
        <div>
          {/* 会话信息 */}
          <Card size="small" style={{ marginBottom: 16 }}>
            <Space direction="vertical" size={4}>
              <Text><strong>会话ID:</strong> {session?.id}</Text>
              <Text><strong>服务器:</strong> {session?.server_name}</Text>
              <Text><strong>用户:</strong> {session?.username}</Text>
              <Text><strong>开始时间:</strong> {session?.start_time ? new Date(session.start_time).toLocaleString() : '-'}</Text>
              <Text><strong>结束时间:</strong> {session?.end_time ? new Date(session.end_time).toLocaleString() : '-'}</Text>
            </Space>
          </Card>

          {/* 控制面板 */}
          <Card size="small" style={{ marginBottom: 16 }}>
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Space>
                <Button
                  type="primary"
                  icon={isPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                  onClick={isPlaying ? pause : play}
                  disabled={!recordingData}
                >
                  {isPlaying ? '暂停' : '播放'}
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={restart}
                  disabled={!recordingData}
                >
                  重新开始
                </Button>
                <Text>播放速度: {playbackSpeed}x</Text>
                <Slider
                  min={0.25}
                  max={3}
                  step={0.25}
                  value={playbackSpeed}
                  onChange={setPlaybackSpeed}
                  style={{ width: 100 }}
                />
                <Text>{formatTime(currentTime)} / {formatTime(duration)}</Text>
              </Space>
              
              <Slider
                min={0}
                max={duration}
                step={0.1}
                value={currentTime}
                onChange={seekTo}
                disabled={!recordingData}
              />
            </Space>
          </Card>

          {/* 终端输出 */}
          <Card size="small">
            <div
              ref={terminalRef}
              style={{
                backgroundColor: '#000',
                color: '#fff',
                padding: '12px',
                borderRadius: '4px',
                minHeight: '400px',
                maxHeight: '600px',
                overflow: 'auto',
                fontFamily: 'Consolas, Monaco, "Courier New", monospace',
              }}
            />
          </Card>
        </div>
      )}
    </Modal>
  );
};

export default SessionReplay;