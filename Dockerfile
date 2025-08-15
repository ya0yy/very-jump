# 构建阶段
FROM golang:1.21-alpine AS backend-builder

WORKDIR /app

# 安装必要的包
RUN apk add --no-cache gcc musl-dev sqlite-dev

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o very-jump cmd/server/main.go

# 前端构建阶段
FROM node:18-alpine AS frontend-builder
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# 运行阶段
FROM tsl0922/ttyd:1.7.7

# 安装运行时依赖
RUN apk --no-cache add ca-certificates sqlite openssh-client sshpass wget

# 安装ttyd
RUN wget -O /usr/local/bin/ttyd https://github.com/tsl0922/ttyd/releases/download/1.7.4/ttyd.x86_64 && \
    chmod +x /usr/local/bin/ttyd

# 创建非root用户
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# 复制构建的二进制文件
COPY --from=backend-builder /app/very-jump .

# 复制前端构建产物
COPY --from=frontend-builder /app/dist /app/web/dist

# 创建数据目录
RUN mkdir -p /data && chown -R appuser:appgroup /data /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 挂载点
VOLUME ["/data"]

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动命令
CMD ["./very-jump"]
