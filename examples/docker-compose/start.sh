#!/bin/bash

# VergeOS Monitoring Stack Startup Script
# Pulls images, builds, and starts the stack, then displays access URLs

set -e

echo "========================================"
echo "VergeOS Monitoring Stack - Starting"
echo "========================================"
echo ""

# Pull latest images
echo "📦 Pulling latest images..."
docker compose pull

echo ""
echo "🔨 Building vergeos-exporter..."
docker compose build --no-cache --pull vergeos-exporter

echo ""
echo "🚀 Starting services..."
docker compose up -d

echo ""
echo "⏳ Waiting for services to be ready..."
sleep 3

# Detect the host IP address
# Try to get the primary network interface IP (works on macOS and Linux)
if command -v ipconfig &> /dev/null; then
    # macOS
    HOST_IP=$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || echo "")
elif command -v hostname &> /dev/null; then
    # Linux/Unix - try hostname -I first, fallback to ip command
    HOST_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || ip route get 1 2>/dev/null | awk '{print $7}' | head -n1 || echo "")
else
    HOST_IP=""
fi

# Check if services are running
if docker compose ps | grep -q "Up"; then
    echo ""
    echo "========================================"
    echo "✅ Services are running!"
    echo "========================================"
    echo ""
    echo "Access your monitoring stack:"
    echo ""
    echo "  📊 Grafana Dashboard:"
    echo "     http://localhost:3000"
    if [ -n "$HOST_IP" ]; then
        echo "     http://${HOST_IP}:3000 (network access)"
    fi
    echo "     Username: admin"
    echo "     Password: (check your .env file)"
    echo ""
    echo "  📈 Prometheus:"
    echo "     http://localhost:9090"
    if [ -n "$HOST_IP" ]; then
        echo "     http://${HOST_IP}:9090 (network access)"
    fi
    echo ""
    echo "  🔧 VergeOS Exporter Metrics:"
    echo "     http://localhost:9888/metrics"
    if [ -n "$HOST_IP" ]; then
        echo "     http://${HOST_IP}:9888/metrics (network access)"
    fi
    echo ""
    echo "========================================"
    echo ""
    echo "💡 Useful commands:"
    echo "  - View logs: docker compose logs -f"
    echo "  - Stop stack: docker compose down"
    echo "  - Check status: docker compose ps"
    echo ""
else
    echo ""
    echo "⚠️  Warning: Some services may not have started correctly"
    echo "   Check status with: docker compose ps"
    echo "   View logs with: docker compose logs"
    echo ""
fi
