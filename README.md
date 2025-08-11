# Very-Jump 轻量级跳板机

一个基于 Go + React + TypeScript 的轻量级跳板机解决方案，专为中小团队设计。

## 特性

- 🚀 **轻量级**: 单容器部署，资源占用少
- 🔒 **安全**: JWT 认证，权限控制，会话录制
- 🌐 **Web 界面**: 现代化的 React + TypeScript + Ant Design 前端
- 📱 **响应式**: 支持移动端访问
- 🐳 **容器化**: 完全 Docker 化部署
- 💾 **便携性**: 数据隔离在挂载目录，易于迁移

## 快速开始

### 使用 Docker Compose（推荐）

1. 克隆项目
```bash
git clone <repository-url>
cd very-jump
```

2. 构建前端
```bash
./scripts/build-frontend.sh
```

3. 启动服务
```bash
docker-compose up -d
```

4. 访问应用
```
http://localhost:8080
```

默认管理员账号：
- 用户名：`admin`
- 密码：`admin`

### 手动部署

1. 构建前端
```bash
chmod +x scripts/build-frontend.sh
./scripts/build-frontend.sh
```

2. 构建后端
```bash
chmod +x scripts/build.sh
./scripts/build.sh
```

3. 创建数据目录
```bash
mkdir -p data
```

4. 启动服务
```bash
./bin/very-jump
```

## 配置

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `PORT` | `8080` | 服务端口 |
| `DATA_DIR` | `/data` | 数据目录 |
| `JWT_SECRET` | `very-jump-secret-key` | JWT 密钥 |
| `JWT_EXPIRY` | `24h` | JWT 过期时间 |
| `SESSION_TIMEOUT` | `30m` | 会话超时时间 |
| `MAX_CONCURRENT_CONN` | `50` | 最大并发连接数 |
| `RECORDING_RETENTION` | `720h` | 录制文件保留时间 |
| `LOG_RETENTION` | `2160h` | 日志保留时间 |

### 数据目录结构

```
/data/
├── very-jump.db        # SQLite 数据库
├── sessions/           # 会话录制文件
│   └── 2024/01/01/
├── config/             # 配置文件
└── logs/               # 应用日志
```

## 功能说明

### 用户管理
- 支持用户创建、编辑、删除
- 角色权限控制（admin/user）
- JWT 认证机制

### 服务器管理
- 支持添加/删除服务器
- 密码或密钥认证
- 服务器分组和标签

### SSH 代理
- Web 终端界面
- 实时命令执行
- 多会话支持
- 终端大小调整

### 会话录制
- 自动录制所有会话
- 支持会话回放
- 录制文件管理

### 权限控制
- 用户-服务器权限映射
- 读写权限控制
- 操作审计日志

## API 文档

### 认证接口

```bash
# 登录
POST /api/v1/auth/login
{
  "username": "admin",
  "password": "admin"
}

# 获取用户信息
GET /api/v1/auth/profile
Authorization: Bearer <token>
```

### 服务器管理

```bash
# 获取服务器列表
GET /api/v1/servers

# 创建服务器（管理员）
POST /api/v1/admin/servers
{
  "name": "测试服务器",
  "host": "192.168.1.100",
  "port": 22,
  "username": "root",
  "auth_type": "password",
  "password": "password"
}
```

### WebSocket 连接

```bash
# SSH 连接
WS /api/v1/ws/ssh/{server_id}
Authorization: Bearer <token>
```

## 开发

### 后端开发

```bash
# 安装依赖
go mod download

# 运行开发服务器
go run cmd/server/main.go

# 运行测试
go test ./...
```

### 前端开发

```bash
# 启动后端开发服务器
go run cmd/server/main.go

# 新开终端，启动前端开发服务器
cd web-app
npm install
npm run dev

# 访问开发环境
# 前端开发服务器: http://localhost:3000
# 后端 API 服务器: http://localhost:8080
```

### 技术栈

**后端**
- Go 1.21+
- Gin Web Framework
- SQLite 数据库
- JWT 认证
- WebSocket 支持

**前端**
- React 19
- TypeScript
- Vite 构建工具
- Ant Design UI 组件库
- Zustand 状态管理
- xterm.js 终端组件
- Axios HTTP 客户端

## 部署建议

### 生产环境

1. **修改默认密码**
```bash
docker-compose exec very-jump ./very-jump admin reset-password
```

2. **配置 HTTPS**
   - 使用 Traefik 或 Nginx 反向代理
   - 配置 SSL 证书

3. **数据备份**
```bash
# 备份数据目录
tar -czf very-jump-backup-$(date +%Y%m%d).tar.gz data/
```

4. **监控**
   - 配置健康检查
   - 监控资源使用
   - 日志聚合

### 安全建议

1. **网络安全**
   - 限制访问源 IP
   - 使用 VPN 或内网访问
   - 定期更新系统和依赖

2. **认证安全**
   - 强密码策略
   - 定期更换 JWT 密钥
   - 启用会话超时

3. **审计安全**
   - 定期检查操作日志
   - 监控异常连接
   - 备份会话录制

## 故障排除

### 常见问题

1. **容器启动失败**
```bash
# 查看日志
docker-compose logs very-jump

# 检查端口占用
netstat -tlnp | grep 8080
```

2. **SSH 连接失败**
   - 检查服务器网络连接
   - 验证认证信息
   - 查看防火墙设置

3. **权限问题**
   - 检查用户-服务器权限配置
   - 验证 JWT Token 有效性

### 日志查看

```bash
# 容器日志
docker-compose logs -f very-jump

# 应用日志
docker-compose exec very-jump tail -f /data/logs/app.log
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

## 支持

如有问题，请创建 Issue 或联系维护者。
