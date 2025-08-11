# 跳板机集成ttyd方案

## 架构设计

```
用户浏览器 → 跳板机前端 → 跳板机后端 → ttyd实例 → 目标服务器
```

## 实现方案

### 方案1: ttyd集成模式

1. **动态启动ttyd进程**
   - 用户点击连接时，后端动态启动ttyd进程
   - 每个SSH连接对应一个ttyd实例
   - 使用不同端口避免冲突

2. **代理模式**
   ```go
   // 启动ttyd进程
   cmd := exec.Command("ttyd", "-p", port, "-o", "ssh", fmt.Sprintf("%s@%s", username, host))
   
   // 代理请求到ttyd
   proxy := httputil.NewSingleHostReverseProxy(ttydURL)
   ```

### 方案2: iframe嵌入模式

1. **前端修改**
   ```tsx
   <iframe 
     src={`http://localhost:${ttydPort}`}
     width="100%" 
     height="100%"
     style={{ border: 'none' }}
   />
   ```

2. **后端API**
   ```
   POST /api/v1/terminal/start/:serverId
   返回: { "port": 7681, "url": "http://localhost:7681" }
   ```

### 方案3: WebSocket代理模式

1. **透明代理**
   - 前端仍连接跳板机WebSocket
   - 后端代理到ttyd的WebSocket
   - 保持现有API不变

## 优势

✅ **成熟稳定** - ttyd经过大量生产环境验证
✅ **功能完整** - 支持复制粘贴、调色板、全屏等
✅ **性能优秀** - C语言实现，资源消耗低
✅ **维护成本低** - 不需要维护终端相关代码

## 实施步骤

1. 安装ttyd二进制文件
2. 修改后端API，集成ttyd启动逻辑
3. 前端改为iframe或代理模式
4. 测试并优化

## Docker集成

```dockerfile
FROM golang:1.21-alpine AS backend-build
# ... 后端构建

FROM node:20-alpine AS frontend-build  
# ... 前端构建

FROM alpine:latest
RUN apk add --no-cache ttyd openssh-client
COPY --from=backend-build /app/very-jump /usr/local/bin/
COPY --from=frontend-build /app/dist /var/www/html/
EXPOSE 8080
CMD ["very-jump"]
```









