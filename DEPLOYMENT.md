# Very-Jump 部署指南

## 快速开始

### 方式一：直接运行（推荐用于开发和测试）

1. **构建项目**
```bash
# 克隆项目（如果需要）
git clone <your-repo-url>
cd very-jump

# 构建
chmod +x scripts/build.sh
./scripts/build.sh
```

2. **创建数据目录**
```bash
mkdir -p data
```

3. **启动服务**
```bash
export DATA_DIR=./data
./bin/very-jump
```

4. **访问应用**
```
浏览器打开: http://localhost:8080
默认账号: admin / admin
```

### 方式二：Docker 部署（推荐用于生产环境）

1. **使用 Docker Compose**
```bash
# 修改 docker-compose.yml 中的环境变量
# 特别是 JWT_SECRET，请在生产环境中更改

docker-compose up -d
```

2. **访问应用**
```
浏览器打开: http://localhost:8080
```

3. **查看日志**
```bash
docker-compose logs -f very-jump
```

## 生产环境配置

### 1. 安全配置

**修改默认密码**
```bash
# 方法1：通过API修改
curl -X PUT http://localhost:8080/api/v1/admin/users/1 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"password": "new_secure_password"}'

# 方法2：直接修改数据库
sqlite3 data/very-jump.db
UPDATE users SET password_hash = 'new_bcrypt_hash' WHERE username = 'admin';
```

**更新 JWT 密钥**
```bash
# 在 docker-compose.yml 或环境变量中设置
export JWT_SECRET="your-very-secure-secret-key-at-least-32-chars"
```

### 2. 反向代理配置

**Nginx 配置示例**
```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    ssl_certificate /path/to/your/cert.pem;
    ssl_certificate_key /path/to/your/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # WebSocket 支持
    location /api/v1/ws/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Traefik 配置示例**
```yaml
# docker-compose.yml 中的 labels
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.very-jump.rule=Host(\`jump.yourdomain.com\`)"
  - "traefik.http.routers.very-jump.tls=true"
  - "traefik.http.routers.very-jump.tls.certresolver=letsencrypt"
```

### 3. 数据备份

**自动备份脚本**
```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/backup/very-jump"
DATA_DIR="/path/to/very-jump/data"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# 备份数据库
sqlite3 "$DATA_DIR/very-jump.db" ".backup $BACKUP_DIR/very-jump_$DATE.db"

# 备份会话录制文件
tar -czf "$BACKUP_DIR/sessions_$DATE.tar.gz" -C "$DATA_DIR" sessions/

# 清理7天前的备份
find "$BACKUP_DIR" -name "*.db" -mtime +7 -delete
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +7 -delete

echo "Backup completed: $DATE"
```

**设置定时备份**
```bash
# 添加到 crontab
crontab -e

# 每天凌晨2点备份
0 2 * * * /path/to/backup.sh >> /var/log/very-jump-backup.log 2>&1
```

### 4. 监控和告警

**健康检查**
```bash
#!/bin/bash
# health-check.sh

HEALTH_URL="http://localhost:8080/health"
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "$HEALTH_URL")

if [ "$RESPONSE" != "200" ]; then
    echo "Very-Jump health check failed: HTTP $RESPONSE"
    # 发送告警通知
    # curl -X POST "your-webhook-url" -d "Very-Jump service is down"
    exit 1
fi

echo "Very-Jump is healthy"
```

**Prometheus 监控**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'very-jump'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'  # 需要添加 metrics 端点
```

## 环境变量说明

| 变量名 | 默认值 | 说明 | 示例 |
|--------|--------|------|------|
| `PORT` | `8080` | 服务端口 | `8080` |
| `DATA_DIR` | `/data` | 数据目录 | `/var/lib/very-jump` |
| `JWT_SECRET` | `very-jump-secret-key` | JWT 签名密钥 | `your-super-secret-key-32-chars` |
| `JWT_EXPIRY` | `24h` | JWT 过期时间 | `24h`, `7d` |
| `SESSION_TIMEOUT` | `30m` | 会话超时时间 | `30m`, `1h` |
| `MAX_CONCURRENT_CONN` | `50` | 最大并发连接数 | `100` |
| `RECORDING_RETENTION` | `720h` | 录制文件保留时间 | `720h` (30天) |
| `LOG_RETENTION` | `2160h` | 日志保留时间 | `2160h` (90天) |

## 故障排除

### 常见问题

1. **端口冲突**
```bash
# 检查端口占用
netstat -tulpn | grep 8080
lsof -i :8080

# 修改端口
export PORT=8081
```

2. **权限问题**
```bash
# 检查数据目录权限
ls -la data/
chmod 755 data/
chown user:group data/
```

3. **数据库锁定**
```bash
# 检查数据库状态
sqlite3 data/very-jump.db "PRAGMA integrity_check;"

# 修复 WAL 模式问题
sqlite3 data/very-jump.db "PRAGMA wal_checkpoint;"
```

4. **内存不足**
```bash
# 检查内存使用
free -h
ps aux | grep very-jump

# 调整 Docker 内存限制
docker-compose.yml 中添加：
deploy:
  resources:
    limits:
      memory: 512M
```

### 日志查看

**应用日志**
```bash
# Docker 方式
docker-compose logs -f very-jump

# 直接运行方式
tail -f data/logs/app.log
```

**数据库查询**
```bash
# 查看用户
sqlite3 data/very-jump.db "SELECT * FROM users;"

# 查看服务器
sqlite3 data/very-jump.db "SELECT * FROM servers;"

# 查看活跃会话
sqlite3 data/very-jump.db "SELECT * FROM sessions WHERE status='active';"

# 查看最近的审计日志
sqlite3 data/very-jump.db "SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT 10;"
```

## 性能优化

### 1. 数据库优化

```bash
# 定期优化数据库
sqlite3 data/very-jump.db "VACUUM;"
sqlite3 data/very-jump.db "ANALYZE;"

# 添加索引（如果需要）
sqlite3 data/very-jump.db "CREATE INDEX idx_sessions_user_id ON sessions(user_id);"
sqlite3 data/very-jump.db "CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);"
```

### 2. 系统调优

```bash
# 增加文件描述符限制
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# 调整内核参数
echo "net.core.somaxconn = 65535" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 65535" >> /etc/sysctl.conf
sysctl -p
```

## 升级指南

### 应用升级

```bash
# 1. 备份数据
./backup.sh

# 2. 停止服务
docker-compose down
# 或
pkill very-jump

# 3. 更新代码
git pull origin main

# 4. 重新构建
./scripts/build.sh
# 或
docker-compose build

# 5. 启动服务
docker-compose up -d
# 或
./bin/very-jump
```

### 数据库迁移

如果有数据库结构变更，需要运行迁移脚本：

```bash
# 查看当前数据库版本
sqlite3 data/very-jump.db "SELECT value FROM settings WHERE key='version';"

# 运行迁移（如果有）
./bin/very-jump --migrate
```

## 安全建议

1. **网络安全**
   - 使用防火墙限制访问端口
   - 启用 HTTPS
   - 使用 VPN 或内网访问

2. **认证安全**
   - 定期更换密码
   - 使用强密码策略
   - 定期轮换 JWT 密钥

3. **系统安全**
   - 定期更新系统和依赖
   - 限制容器权限
   - 启用审计日志

4. **数据安全**
   - 定期备份数据
   - 加密敏感数据
   - 设置数据保留策略

---

如有问题，请查看项目 README.md 或创建 Issue。
