import { sessionAPI } from './api';

export class SessionHeartbeat {
  private sessionId: string;
  private intervalId: number | null = null;
  private readonly heartbeatInterval = 60000; // 1分钟发送一次心跳
  private isActive = false;

  constructor(sessionId: string) {
    this.sessionId = sessionId;
  }

  // 开始心跳
  start(): void {
    if (this.isActive) {
      return;
    }

    this.isActive = true;
    
    // 立即发送一次心跳
    this.sendHeartbeat();
    
    // 设置定时器
    this.intervalId = window.setInterval(() => {
      this.sendHeartbeat();
    }, this.heartbeatInterval);

    // 监听页面可见性变化
    document.addEventListener('visibilitychange', this.handleVisibilityChange);
    
    // 监听页面卸载事件
    window.addEventListener('beforeunload', this.handleBeforeUnload);
    window.addEventListener('unload', this.handleUnload);
    
    console.log(`Session heartbeat started for session: ${this.sessionId}`);
  }

  // 停止心跳
  stop(): void {
    if (!this.isActive) {
      return;
    }

    this.isActive = false;
    
    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }

    // 移除事件监听器
    document.removeEventListener('visibilitychange', this.handleVisibilityChange);
    window.removeEventListener('beforeunload', this.handleBeforeUnload);
    window.removeEventListener('unload', this.handleUnload);
    
    console.log(`Session heartbeat stopped for session: ${this.sessionId}`);
  }

  // 发送心跳
  private sendHeartbeat = async (): Promise<void> => {
    if (!this.isActive) {
      return;
    }

    try {
      await sessionAPI.sendHeartbeat(this.sessionId);
      console.log(`Heartbeat sent for session: ${this.sessionId}`);
    } catch (error) {
      console.warn(`Failed to send heartbeat for session ${this.sessionId}:`, error);
      
      // 如果会话不存在或无权限，停止心跳
      if (error instanceof Error) {
        const errorMsg = error.message.toLowerCase();
        if (errorMsg.includes('not found') || errorMsg.includes('forbidden')) {
          this.stop();
        }
      }
    }
  };

  // 处理页面可见性变化
  private handleVisibilityChange = (): void => {
    if (document.hidden) {
      // 页面隐藏时，可以减少心跳频率或暂停
      console.log('Page hidden, session heartbeat continues');
    } else {
      // 页面重新可见时，立即发送心跳
      this.sendHeartbeat();
      console.log('Page visible, sending immediate heartbeat');
    }
  };

  // 处理页面卸载前事件
  private handleBeforeUnload = (): void => {
    // 尝试发送最后一次心跳（但不能保证成功）
    this.sendFinalHeartbeat();
  };

  // 处理页面卸载事件
  private handleUnload = (): void => {
    this.sendFinalHeartbeat();
  };

  // 发送最终心跳（同步方式）
  private sendFinalHeartbeat(): void {
    if (!this.isActive) {
      return;
    }

    try {
      // 使用sendBeacon API发送最后的请求
      if (navigator.sendBeacon) {
        const data = new FormData();
        data.append('sessionId', this.sessionId);
        
        navigator.sendBeacon(`/api/v1/sessions/${this.sessionId}/heartbeat`, data);
        console.log(`Final heartbeat sent via beacon for session: ${this.sessionId}`);
      } else {
        // 降级到同步XMLHttpRequest
        const xhr = new XMLHttpRequest();
        xhr.open('POST', `/api/v1/sessions/${this.sessionId}/heartbeat`, false);
        xhr.setRequestHeader('Content-Type', 'application/json');
        
        const token = localStorage.getItem('token');
        if (token) {
          xhr.setRequestHeader('Authorization', `Bearer ${token}`);
        }
        
        xhr.send();
        console.log(`Final heartbeat sent via XHR for session: ${this.sessionId}`);
      }
    } catch (error) {
      console.warn(`Failed to send final heartbeat for session ${this.sessionId}:`, error);
    }
  }

  // 更新会话ID
  updateSessionId(newSessionId: string): void {
    this.sessionId = newSessionId;
  }

  // 检查是否活跃
  isHeartbeatActive(): boolean {
    return this.isActive;
  }
}

// 全局心跳管理器
class HeartbeatManager {
  private heartbeats: Map<string, SessionHeartbeat> = new Map();

  // 开始会话心跳
  startHeartbeat(sessionId: string): void {
    if (this.heartbeats.has(sessionId)) {
      return;
    }

    const heartbeat = new SessionHeartbeat(sessionId);
    this.heartbeats.set(sessionId, heartbeat);
    heartbeat.start();
  }

  // 停止会话心跳
  stopHeartbeat(sessionId: string): void {
    const heartbeat = this.heartbeats.get(sessionId);
    if (heartbeat) {
      heartbeat.stop();
      this.heartbeats.delete(sessionId);
    }
  }

  // 停止所有心跳
  stopAllHeartbeats(): void {
    this.heartbeats.forEach((heartbeat) => {
      heartbeat.stop();
    });
    this.heartbeats.clear();
  }

  // 获取活跃的心跳数量
  getActiveHeartbeatCount(): number {
    return this.heartbeats.size;
  }
}

// 导出全局实例
export const heartbeatManager = new HeartbeatManager();