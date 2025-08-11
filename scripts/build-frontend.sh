#!/bin/bash

set -e

echo "=== Very-Jump 前端构建脚本 ==="

# 检查 Node.js 是否安装
if ! command -v node &> /dev/null; then
    echo "❌ Node.js 未安装，请先安装 Node.js"
    exit 1
fi

# 检查 npm 是否安装
if ! command -v npm &> /dev/null; then
    echo "❌ npm 未安装，请先安装 npm"
    exit 1
fi

echo "✅ Node.js 版本: $(node --version)"
echo "✅ npm 版本: $(npm --version)"

# 进入前端目录
cd web-app

echo "📦 安装前端依赖..."
npm install

echo "🏗️  构建前端应用..."
npm run build

echo "🔄 复制构建文件到 Go 项目..."
# 构建文件已经通过 vite.config.ts 配置输出到 ../web/dist

echo "✅ 前端构建完成！"
echo ""
echo "📁 构建文件位置: web/dist/"
echo "🚀 现在可以运行 Go 后端服务器来查看完整应用"









