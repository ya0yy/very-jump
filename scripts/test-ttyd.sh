#!/bin/bash

# 测试ttyd集成脚本

set -e

echo "=== 测试ttyd集成 ==="

# 检查ttyd是否已安装
if ! command -v ttyd &> /dev/null; then
    echo "ttyd未安装，正在安装..."
    
    # 根据操作系统安装ttyd
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if command -v brew &> /dev/null; then
            brew install ttyd
        else
            echo "请先安装Homebrew或手动安装ttyd"
            exit 1
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        echo "正在下载ttyd二进制文件..."
        wget -O /tmp/ttyd https://github.com/tsl0922/ttyd/releases/download/1.7.4/ttyd.x86_64
        chmod +x /tmp/ttyd
        sudo mv /tmp/ttyd /usr/local/bin/ttyd
    else
        echo "不支持的操作系统: $OSTYPE"
        exit 1
    fi
fi

# 检查sshpass是否已安装
if ! command -v sshpass &> /dev/null; then
    echo "sshpass未安装，正在安装..."
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install hudochenkov/sshpass/sshpass
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y sshpass
        elif command -v yum &> /dev/null; then
            sudo yum install -y sshpass
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y sshpass
        else
            echo "无法自动安装sshpass，请手动安装"
            exit 1
        fi
    fi
fi

echo "检查依赖完成："
echo "- ttyd版本: $(ttyd --version)"
echo "- sshpass已安装: $(command -v sshpass)"

# 编译Go后端
echo ""
echo "=== 编译Go后端 ==="
cd "$(dirname "$0")/.."
go mod tidy
go build -o bin/very-jump cmd/server/main.go

echo "编译完成：bin/very-jump"

# 创建测试数据目录
echo ""
echo "=== 准备测试环境 ==="
mkdir -p data/temp_keys
mkdir -p data/sessions

echo "测试环境准备完成"

# 启动测试
echo ""
echo "=== 启动测试服务器 ==="
echo "请在浏览器中访问 http://localhost:8080 测试ttyd集成"
echo "默认管理员账户: admin / admin123"
echo ""
echo "按 Ctrl+C 停止服务器"

export DATA_DIR="$(pwd)/data"
export PORT="8080"
export JWT_SECRET="test-secret-key"

./bin/very-jump








