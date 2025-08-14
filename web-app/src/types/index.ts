// API 响应类型定义

export interface User {
  id: number;
  username: string;
  role: 'admin' | 'user';
  created_at: string;
  updated_at: string;
}

export interface Server {
  id: number;
  name: string;
  host: string;
  port: number;
  username: string;
  auth_type: 'password' | 'key' | 'credential';
  credential_id?: number;
  credential_name?: string;
  description: string;
  tags: string[];
  last_login_time?: string;
  status?: 'available' | 'unavailable' | 'checking';
  created_at: string;
  updated_at: string;
}

export interface Session {
  id: string;
  user_id: number;
  server_id: number;
  start_time: string;
  end_time?: string;
  status: 'active' | 'closed' | 'error';
  client_ip: string;
  recording_file: string;
  username?: string;
  server_name?: string;
}

export interface AuditLog {
  id: number;
  user_id: number;
  action: string;
  resource_type: string;
  resource_id: string;
  details: string;
  ip_address: string;
  user_agent: string;
  created_at: string;
  username?: string;
}

// API 请求/响应类型

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
  expires_at: string;
}

export interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

export interface ServerListResponse {
  servers: Server[];
  total: number;
}

export interface SessionListResponse {
  sessions: Session[];
  total: number;
}

export interface Credential {
  id: number;
  name: string;
  type: 'password' | 'key';
  username: string;
  created_at: string;
  updated_at: string;
}

export interface CredentialCreateRequest {
  name: string;
  type: 'password' | 'key';
  username: string;
  password?: string;
  private_key?: string;
  key_password?: string;
}

export interface ServerCreateRequest {
  name: string;
  host: string;
  port: number;
  username: string;
  auth_type: 'password' | 'key' | 'credential';
  password?: string;
  private_key?: string;
  credential_id?: number;
  description: string;
  tags: string[];
}

export interface UserCreateRequest {
  username: string;
  password: string;
  role: 'admin' | 'user';
}

// 应用状态类型

export interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  token: string | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  checkAuth: () => void;
}

export interface AppState {
  servers: Server[];
  sessions: Session[];
  activeSessions: number;
  loading: boolean;
  error: string | null;
  setServers: (servers: Server[]) => void;
  updateServerStatus: (serverId: number, status: string) => void;
  setSessions: (sessions: Session[]) => void;
  setActiveSessions: (count: number) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

// 组件 Props 类型

export interface ServerCardProps {
  server: Server;
  onConnect: (server: Server) => void;
  onEdit?: (server: Server) => void;
  onDelete?: (server: Server) => void;
}

export interface SessionItemProps {
  session: Session;
  onClose?: (sessionId: string) => void;
  onView?: (sessionId: string) => void;
}

export interface TerminalProps {
  serverId: number;
  serverName: string;
  onClose: () => void;
}

// WebSocket 消息类型

export interface WebSocketMessage {
  type: 'input' | 'output' | 'resize' | 'close' | 'stdout' | 'stderr';
  data: string | ResizeData;
}

export interface ResizeData {
  cols: number;
  rows: number;
}
