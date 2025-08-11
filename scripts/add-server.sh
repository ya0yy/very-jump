#!/bin/bash

# Very-Jump 服务器添加脚本

echo "=== Very-Jump 服务器管理工具 ==="
echo ""

# 检查服务是否运行
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "❌ Very-Jump 服务未运行，请先启动服务"
    exit 1
fi

echo "✅ 服务运行正常"

# 获取登录 Token
echo "🔐 正在获取认证 Token..."
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin"}' | \
  grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败，请检查用户名密码"
    exit 1
fi

echo "✅ 认证成功"

# 交互式添加服务器
echo ""
echo "📝 请输入服务器信息："

read -p "服务器名称: " SERVER_NAME
read -p "服务器地址: " SERVER_HOST
read -p "SSH 端口 (默认22): " SERVER_PORT
SERVER_PORT=${SERVER_PORT:-22}
read -p "SSH 用户名: " SERVER_USERNAME

echo "选择认证方式:"
echo "1) 密码认证"
echo "2) 密钥认证"
read -p "请选择 (1/2): " AUTH_CHOICE

if [ "$AUTH_CHOICE" = "1" ]; then
    read -s -p "SSH 密码: " SERVER_PASSWORD
    echo ""
    read -p "服务器描述 (可选): " SERVER_DESC
    
    # 添加服务器
    echo ""
    echo "🚀 正在添加服务器..."
    RESULT=$(curl -s -X POST http://localhost:8080/api/v1/admin/servers \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" \
      -d "{
        \"name\": \"$SERVER_NAME\",
        \"host\": \"$SERVER_HOST\",
        \"port\": $SERVER_PORT,
        \"username\": \"$SERVER_USERNAME\",
        \"auth_type\": \"password\",
        \"password\": \"$SERVER_PASSWORD\",
        \"description\": \"$SERVER_DESC\"
      }")
      
elif [ "$AUTH_CHOICE" = "2" ]; then
    echo "请输入私钥文件路径:"
    read -p "私钥路径: " KEY_PATH
    
    if [ ! -f "$KEY_PATH" ]; then
        echo "❌ 私钥文件不存在: $KEY_PATH"
        exit 1
    fi
    
    PRIVATE_KEY=$(cat "$KEY_PATH" | sed ':a;N;$!ba;s/\n/\\n/g')
    read -p "服务器描述 (可选): " SERVER_DESC
    
    # 添加服务器
    echo ""
    echo "🚀 正在添加服务器..."
    RESULT=$(curl -s -X POST http://localhost:8080/api/v1/admin/servers \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" \
      -d "{
        \"name\": \"$SERVER_NAME\",
        \"host\": \"$SERVER_HOST\",
        \"port\": $SERVER_PORT,
        \"username\": \"$SERVER_USERNAME\",
        \"auth_type\": \"key\",
        \"private_key\": \"$PRIVATE_KEY\",
        \"description\": \"$SERVER_DESC\"
      }")
else
    echo "❌ 无效选择"
    exit 1
fi

# 检查结果
if echo "$RESULT" | grep -q '"id"'; then
    echo "✅ 服务器添加成功！"
    echo ""
    echo "服务器信息:"
    echo "$RESULT" | python3 -m json.tool 2>/dev/null || echo "$RESULT"
else
    echo "❌ 服务器添加失败："
    echo "$RESULT"
fi

echo ""
echo "💡 提示: 现在可以在 Web 界面 (http://localhost:8080) 中看到新添加的服务器"



