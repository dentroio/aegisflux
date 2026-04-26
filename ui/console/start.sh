#!/bin/bash

# AegisFlux Console Startup Script
echo "🚀 Starting AegisFlux Console..."

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 18+ and try again."
    exit 1
fi

# Check Node.js version
NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 18 ]; then
    echo "❌ Node.js version 18+ is required. Current version: $(node -v)"
    exit 1
fi

echo "✅ Node.js $(node -v) detected"

# Check if dependencies are installed
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
    if [ $? -ne 0 ]; then
        echo "❌ Failed to install dependencies"
        exit 1
    fi
    echo "✅ Dependencies installed successfully"
else
    echo "✅ Dependencies already installed"
fi

# Check backend services
echo "🔍 Checking backend services..."

# Function to check service health
check_service() {
    local service=$1
    local port=$2
    local name=$3
    
    if curl -s "http://localhost:$port/healthz" > /dev/null 2>&1; then
        echo "✅ $name (port $port) is running"
        return 0
    else
        echo "⚠️  $name (port $port) is not responding"
        return 1
    fi
}

# Check all backend services
check_service "actions-api" 8083 "Actions API"
check_service "bpf-registry" 8090 "BPF Registry"
check_service "decision" 8087 "Decision Engine"
check_service "orchestrator" 8084 "Orchestrator"

echo ""
echo "🎯 Starting development server..."
echo "📱 Console will be available at: http://127.0.0.1:3030"
echo "🔗 Backend services should be running on:"
echo "   • Actions API: http://localhost:8083"
echo "   • BPF Registry: http://localhost:8090"
echo "   • Decision Engine: http://localhost:8087"
echo "   • Orchestrator: http://localhost:8084"
echo ""

# Start the development server
npm run dev
