# Very-Jump 轻量级跳板机技术架构

## 架构概述

基于用户需求设计的轻量级、可移植的跳板机解决方案。

### 核心设计原则
- **轻量级**: 单个 Docker 容器部署
- **可移植**: 仅需挂载一个数据目录
- **简单**: 专注核心功能，避免过度设计
- **可靠**: 稳定的 SSH 代理和会话管理

## 技术栈选择

### 后端技术栈
- **语言**: Go 1.21+
- **Web框架**: Gin（轻量级，性能好）
- **数据库**: SQLite（文件数据库，便于迁移）
- **SSH库**: golang.org/x/crypto/ssh
- **WebSocket**: gorilla/websocket（实时终端）
- **会话录制**: 自研轻量级录制器

### 前端技术栈
- **框架**: React 18 + TypeScript
- **UI库**: Ant Design
- **终端**: xterm.js + xterm-addon-fit
- **HTTP客户端**: axios
- **状态管理**: Zustand
- **构建工具**: Vite

### 部署技术栈
- **容器**: Docker（单容器部署）
- **反向代理**: 内置 HTTP 服务器
- **数据存储**: 挂载目录 `/data`

## 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────┐
│                Docker Container                      │
│  ┌─────────────────┐    ┌─────────────────────────┐  │
│  │   Frontend      │    │      Backend            │  │
│  │   (React SPA)   │    │      (Go Gin)           │  │
│  │                 │    │                         │  │
│  │  ┌───────────┐  │    │  ┌─────────────────┐    │  │
│  │  │  Web UI   │  │◄──►│  │   HTTP API      │    │  │
│  │  └───────────┘  │    │  └─────────────────┘    │  │
│  │  ┌───────────┐  │    │  ┌─────────────────┐    │  │
│  │  │ Terminal  │  │◄──►│  │   WebSocket     │    │  │
│  │  │ (xterm.js)│  │    │  │   SSH Proxy     │    │  │
│  │  └───────────┘  │    │  └─────────────────┘    │  │
│  └─────────────────┘    │  ┌─────────────────┐    │  │
│                         │  │   SQLite DB     │    │  │
│                         │  │   Session Rec   │    │  │
│                         │  └─────────────────┘    │  │
│                         └─────────────────────────┘  │
└─────────────────────────────────────────────────────┘
                              │
                              ▼
                         /data (挂载目录)
                     ┌─────────────────┐
                     │ very-jump.db    │
                     │ sessions/       │
                     │ config/         │
                     │ logs/           │
                     └─────────────────┘
```

### 核心组件设计

#### 1. HTTP API 服务 (Go Gin)
```go
// 主要路由设计
/api/v1/
├── auth/
│   ├── POST /login     // 用户登录
│   ├── POST /logout    // 用户登出
│   └── GET  /profile   // 获取用户信息
├── servers/
│   ├── GET    /        // 获取服务器列表
│   ├── POST   /        // 添加服务器
│   ├── PUT    /:id     // 更新服务器
│   └── DELETE /:id     // 删除服务器
├── sessions/
│   ├── GET    /        // 获取会话列表
│   ├── POST   /        // 创建新会话
│   └── GET    /:id/playback // 会话回放
└── users/
    ├── GET    /        // 获取用户列表
    ├── POST   /        // 添加用户
    ├── PUT    /:id     // 更新用户
    └── DELETE /:id     // 删除用户
```

#### 2. WebSocket SSH 代理
```go
// WebSocket 处理流程
Client(xterm.js) ◄─► WebSocket Server ◄─► SSH Client ◄─► Target Server
                         │
                         ▼
                    Session Recorder
```

#### 3. 数据模型设计

```sql
-- 用户表
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 服务器表
CREATE TABLE servers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER DEFAULT 22,
    username VARCHAR(100) NOT NULL,
    auth_type VARCHAR(20) DEFAULT 'password', -- password, key
    password VARCHAR(255),
    private_key TEXT,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 用户服务器权限表
CREATE TABLE user_server_permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    server_id INTEGER NOT NULL,
    permission VARCHAR(20) DEFAULT 'read', -- read, write, admin
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (server_id) REFERENCES servers(id),
    UNIQUE(user_id, server_id)
);

-- 会话表
CREATE TABLE sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id INTEGER NOT NULL,
    server_id INTEGER NOT NULL,
    start_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    end_time DATETIME,
    status VARCHAR(20) DEFAULT 'active', -- active, closed, error
    client_ip VARCHAR(45),
    recording_file VARCHAR(255),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (server_id) REFERENCES servers(id)
);

-- 操作日志表
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(100),
    details TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

## 目录结构

```
very-jump/
├── cmd/
│   └── server/
│       └── main.go              // 应用入口
├── internal/
│   ├── api/                     // HTTP API 处理器
│   │   ├── auth.go
│   │   ├── servers.go
│   │   ├── sessions.go
│   │   └── users.go
│   ├── config/                  // 配置管理
│   │   └── config.go
│   ├── database/                // 数据库操作
│   │   ├── db.go
│   │   ├── migrations.go
│   │   └── models/
│   │       ├── user.go
│   │       ├── server.go
│   │       └── session.go
│   ├── middleware/              // 中间件
│   │   ├── auth.go
│   │   ├── cors.go
│   │   └── logging.go
│   ├── services/                // 业务逻辑
│   │   ├── auth_service.go
│   │   ├── ssh_service.go
│   │   ├── session_service.go
│   │   └── recording_service.go
│   └── websocket/               // WebSocket 处理
│       ├── handler.go
│       └── ssh_proxy.go
├── web/                         // 前端代码
│   ├── public/
│   ├── src/
│   │   ├── components/
│   │   │   ├── Terminal.tsx
│   │   │   ├── ServerList.tsx
│   │   │   └── SessionList.tsx
│   │   ├── pages/
│   │   │   ├── Login.tsx
│   │   │   ├── Dashboard.tsx
│   │   │   └── Settings.tsx
│   │   ├── services/
│   │   │   └── api.ts
│   │   └── App.tsx
│   ├── package.json
│   └── vite.config.ts
├── scripts/
│   └── build.sh                 // 构建脚本
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

## 核心功能设计

### 1. SSH 代理连接流程

```
1. 用户在 Web 界面选择服务器
2. 前端通过 WebSocket 连接到后端
3. 后端验证用户权限
4. 建立 SSH 连接到目标服务器
5. 创建会话记录
6. 开始会话录制
7. 代理所有 SSH 流量
8. 记录操作日志
```

### 2. 会话录制机制

```go
type SessionRecorder struct {
    sessionID string
    file      *os.File
    startTime time.Time
}

type RecordEvent struct {
    Timestamp int64  `json:"timestamp"`
    Type      string `json:"type"`     // input, output
    Data      string `json:"data"`
}
```

### 3. 权限控制模型

```
角色层级:
- admin: 管理所有用户和服务器
- user: 只能访问分配的服务器

权限检查流程:
1. JWT Token 验证
2. 用户角色检查
3. 服务器访问权限检查
4. 操作权限检查
```

## 容器化部署

### Dockerfile
```dockerfile
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o very-jump cmd/server/main.go

FROM node:18-alpine AS frontend-builder
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /app
COPY --from=backend-builder /app/very-jump .
COPY --from=frontend-builder /app/dist ./web/dist
EXPOSE 8080
VOLUME ["/data"]
CMD ["./very-jump"]
```

### 数据持久化

```
/data/
├── very-jump.db        // SQLite 数据库文件
├── sessions/           // 会话录制文件
│   ├── 2024/01/01/
│   └── ...
├── config/             // 配置文件
│   └── app.yml
└── logs/               // 应用日志
    ├── app.log
    └── audit.log
```

## 性能考虑

### 1. 并发连接
- 使用 Go 协程处理每个 WebSocket 连接
- 连接池管理 SSH 连接
- 目标：支持 50+ 并发会话

### 2. 会话录制优化
- 异步写入录制文件
- 按日期分目录存储
- 可配置录制保留时间

### 3. 数据库优化
- SQLite WAL 模式
- 适当的索引设计
- 定期清理旧日志

## 安全考虑

### 1. 认证安全
- JWT Token 有效期控制
- 密码 bcrypt 加密存储
- 会话超时机制

### 2. SSH 安全
- 不缓存 SSH 密码
- 支持 SSH 密钥认证
- 连接超时控制

### 3. 审计安全
- 完整的操作日志
- 会话录制文件完整性
- 敏感信息脱敏

## 开发计划

### Phase 1: 核心功能 (2-3周)
- [ ] 基础项目结构搭建
- [ ] 用户认证系统
- [ ] 服务器管理 CRUD
- [ ] 基础 SSH 代理功能
- [ ] 简单的 Web 界面

### Phase 2: 高级功能 (2-3周)
- [ ] 会话录制和回放
- [ ] 用户权限管理
- [ ] 操作审计日志
- [ ] 完善的 Web 界面

### Phase 3: 生产优化 (1-2周)
- [ ] 性能优化
- [ ] 错误处理完善
- [ ] 容器化部署
- [ ] 文档完善

---

**下一步**: 基于此架构创建详细的 PRD 文档并开始开发。
