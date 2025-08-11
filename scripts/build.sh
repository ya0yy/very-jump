#!/bin/bash

set -e

echo "Building Very-Jump..."

# 构建后端
echo "Building backend..."
CGO_ENABLED=1 go build -o bin/very-jump cmd/server/main.go

# 构建前端（后续添加）
# echo "Building frontend..."
# cd web
# npm install
# npm run build
# cd ..

echo "Build completed successfully!"

# 设置权限
chmod +x bin/very-jump

echo "Binary available at: bin/very-jump"
