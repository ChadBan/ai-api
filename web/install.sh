#!/bin/bash

echo "🚀 Installing AI Model Scheduler Web Frontend..."

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 18+ first."
    exit 1
fi

echo "✅ Node.js version: $(node -v)"

# 进入 web 目录
cd "$(dirname "$0")"

# 安装依赖
echo "📦 Installing dependencies..."
npm install --registry=https://registry.npmmirror.com

# 构建
echo "🔨 Building frontend..."
npm run build

echo "✅ Frontend build completed!"
echo ""
echo "To start the server:"
echo "  cd /usr/local/go-path/ai-api"
echo "  go run cmd/server/main.go"
echo ""
echo "Then visit: http://localhost:8080"
