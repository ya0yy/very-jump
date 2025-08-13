import axios, { type AxiosResponse } from 'axios';
import type { 
  LoginRequest, 
  LoginResponse, 
  User, 
  ServerListResponse, 
  SessionListResponse,
  ServerCreateRequest,
  UserCreateRequest,
  Server,
  Session,
  AuditLog
} from '../types';

// 创建 axios 实例
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
});

// 请求拦截器 - 自动添加 token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// 响应拦截器 - 处理错误
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Token 过期，清除本地存储并跳转登录
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      // 清除 Zustand 持久化的认证状态，防止错误的自动登录状态
      localStorage.removeItem('auth-storage');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// 认证相关 API
export const authAPI = {
  login: async (credentials: LoginRequest): Promise<LoginResponse> => {
    const response: AxiosResponse<LoginResponse> = await api.post('/auth/login', credentials);
    return response.data;
  },

  logout: async (): Promise<void> => {
    await api.post('/auth/logout');
  },

  getProfile: async (): Promise<User> => {
    const response: AxiosResponse<User> = await api.get('/auth/profile');
    return response.data;
  },
};

// 服务器管理 API
export const serverAPI = {
  getServers: async (): Promise<ServerListResponse> => {
    const response: AxiosResponse<ServerListResponse> = await api.get('/servers');
    return response.data;
  },

  getServer: async (id: number): Promise<Server> => {
    const response: AxiosResponse<Server> = await api.get(`/servers/${id}`);
    return response.data;
  },

  createServer: async (data: ServerCreateRequest): Promise<Server> => {
    const response: AxiosResponse<Server> = await api.post('/admin/servers', data);
    return response.data;
  },

  updateServer: async (id: number, data: Partial<ServerCreateRequest>): Promise<Server> => {
    const response: AxiosResponse<Server> = await api.put(`/admin/servers/${id}`, data);
    return response.data;
  },

  deleteServer: async (id: number): Promise<void> => {
    await api.delete(`/admin/servers/${id}`);
  },

  checkServerStatus: async (id: number): Promise<{ server_id: number; status: string }> => {
    const response = await api.get(`/servers/${id}/status`);
    return response.data;
  },

  testConnection: async (id: number): Promise<{ success: boolean; message: string }> => {
    const response = await api.post(`/admin/servers/${id}/test`);
    return response.data;
  },
};

// 会话管理 API
export const sessionAPI = {
  getSessions: async (): Promise<SessionListResponse> => {
    const response: AxiosResponse<SessionListResponse> = await api.get('/sessions');
    return response.data;
  },

  getSession: async (id: string): Promise<Session> => {
    const response: AxiosResponse<Session> = await api.get(`/sessions/${id}`);
    return response.data;
  },

  closeSession: async (id: string): Promise<void> => {
    await api.post(`/sessions/${id}/close`);
  },

  getActiveSessions: async (): Promise<{ active_sessions: number }> => {
    const response = await api.get('/sessions/active');
    return response.data;
  },

  getReplayInfo: async (id: string): Promise<{ 
    session: Session;
    has_recording: boolean; 
    recording_size: number; 
  }> => {
    const response = await api.get(`/sessions/${id}/replay-info`);
    return response.data;
  },

  getRecordingFile: async (id: string): Promise<Blob> => {
    const response = await api.get(`/sessions/${id}/replay`, {
      responseType: 'blob'
    });
    return response.data;
  },

  sendHeartbeat: async (id: string): Promise<void> => {
    await api.post(`/sessions/${id}/heartbeat`);
  },
};

// 用户管理 API (管理员)
export const userAPI = {
  getUsers: async (): Promise<{ users: User[]; total: number }> => {
    const response = await api.get('/admin/users');
    return response.data;
  },

  getUser: async (id: number): Promise<User> => {
    const response: AxiosResponse<User> = await api.get(`/admin/users/${id}`);
    return response.data;
  },

  createUser: async (data: UserCreateRequest): Promise<User> => {
    const response: AxiosResponse<User> = await api.post('/admin/users', data);
    return response.data;
  },

  updateUser: async (id: number, data: Partial<UserCreateRequest>): Promise<User> => {
    const response: AxiosResponse<User> = await api.put(`/admin/users/${id}`, data);
    return response.data;
  },

  deleteUser: async (id: number): Promise<void> => {
    await api.delete(`/admin/users/${id}`);
  },
};

// 审计日志 API
export const auditAPI = {
  getLogs: async (params?: {
    action?: string;
    start_date?: string;
    end_date?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ logs: AuditLog[]; total: number }> => {
    const response = await api.get('/audit-logs', { params });
    return response.data;
  },
};

// 系统 API
export const systemAPI = {
  getHealth: async (): Promise<{ status: string; version: string }> => {
    const response = await api.get('/health');
    return response.data;
  },

  getStats: async (): Promise<{
    total_servers: number;
    active_sessions: number;
    total_users: number;
    total_sessions: number;
  }> => {
    const response = await api.get('/admin/stats');
    return response.data;
  },
};

export default api;
