# VergeOS Monitoring Stack Startup Script for Windows
# Pulls images, builds, and starts the stack, then displays access URLs

# Stop on errors
$ErrorActionPreference = "Stop"

Write-Host "========================================"
Write-Host "VergeOS Monitoring Stack - Starting"
Write-Host "========================================"
Write-Host ""

# Pull latest images
Write-Host "📦 Pulling latest images..."
docker compose pull

Write-Host ""
Write-Host "🔨 Building vergeos-exporter..."
docker compose build --no-cache --pull vergeos-exporter

Write-Host ""
Write-Host "🚀 Starting services..."
docker compose up -d

Write-Host ""
Write-Host "⏳ Waiting for services to be ready..."
Start-Sleep -Seconds 3

# Detect the host IP address
# Get the primary network adapter's IPv4 address
$HostIP = ""
try {
    $HostIP = (Get-NetIPAddress -AddressFamily IPv4 -InterfaceAlias "Ethernet*", "Wi-Fi*" -ErrorAction SilentlyContinue |
               Where-Object {$_.IPAddress -notlike "169.254.*" -and $_.IPAddress -notlike "127.*"} |
               Select-Object -First 1).IPAddress
} catch {
    # Fallback: try alternative method
    try {
        $HostIP = (Get-NetIPConfiguration |
                   Where-Object {$_.IPv4DefaultGateway -ne $null -and $_.NetAdapter.Status -eq "Up"} |
                   Select-Object -First 1).IPv4Address.IPAddress
    } catch {
        $HostIP = ""
    }
}

# Check if services are running
$runningServices = docker compose ps
if ($runningServices -match "Up") {
    Write-Host ""
    Write-Host "========================================"
    Write-Host "✅ Services are running!" -ForegroundColor Green
    Write-Host "========================================"
    Write-Host ""
    Write-Host "Access your monitoring stack:"
    Write-Host ""
    Write-Host "  📊 Grafana Dashboard:"
    Write-Host "     http://localhost:3000"
    if ($HostIP) {
        Write-Host "     http://${HostIP}:3000 (network access)" -ForegroundColor Cyan
    }
    Write-Host "     Username: admin"
    Write-Host "     Password: (check your .env file)"
    Write-Host ""
    Write-Host "  📈 Prometheus:"
    Write-Host "     http://localhost:9090"
    if ($HostIP) {
        Write-Host "     http://${HostIP}:9090 (network access)" -ForegroundColor Cyan
    }
    Write-Host ""
    Write-Host "  🔧 VergeOS Exporter Metrics:"
    Write-Host "     http://localhost:9888/metrics"
    if ($HostIP) {
        Write-Host "     http://${HostIP}:9888/metrics (network access)" -ForegroundColor Cyan
    }
    Write-Host ""
    Write-Host "========================================"
    Write-Host ""
    Write-Host "💡 Useful commands:"
    Write-Host "  - View logs: docker compose logs -f"
    Write-Host "  - Stop stack: docker compose down"
    Write-Host "  - Check status: docker compose ps"
    Write-Host ""
} else {
    Write-Host ""
    Write-Host "⚠️  Warning: Some services may not have started correctly" -ForegroundColor Yellow
    Write-Host "   Check status with: docker compose ps"
    Write-Host "   View logs with: docker compose logs"
    Write-Host ""
}
