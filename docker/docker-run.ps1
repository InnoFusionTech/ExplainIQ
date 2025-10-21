# ExplainIQ Docker Compose Runner
# This script provides easy commands to manage the ExplainIQ platform with Docker Compose

param(
    [Parameter(Position=0)]
    [ValidateSet("up", "down", "build", "logs", "status", "restart", "dev")]
    [string]$Command = "up",
    
    [Parameter(Position=1)]
    [string]$Service = ""
)

Write-Host "🐳 ExplainIQ Docker Compose Manager" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan

switch ($Command) {
    "up" {
        Write-Host "🚀 Starting ExplainIQ services..." -ForegroundColor Green
        if ($Service) {
            docker-compose up -d $Service
        } else {
            docker-compose up -d
        }
        Write-Host "✅ Services started! Access the application at:" -ForegroundColor Green
        Write-Host "   • Frontend (Go): http://localhost:8085" -ForegroundColor White
        Write-Host "   • Frontend (Next.js): http://localhost:3000" -ForegroundColor White
        Write-Host "   • API: http://localhost:8080" -ForegroundColor White
    }
    
    "dev" {
        Write-Host "🔧 Starting ExplainIQ in development mode..." -ForegroundColor Yellow
        docker-compose -f docker-compose.dev.yml up -d
        Write-Host "✅ Development services started!" -ForegroundColor Green
        Write-Host "   • Frontend (Go): http://localhost:8085" -ForegroundColor White
        Write-Host "   • Frontend (Next.js): http://localhost:3000" -ForegroundColor White
        Write-Host "   • API: http://localhost:8080" -ForegroundColor White
    }
    
    "down" {
        Write-Host "🛑 Stopping ExplainIQ services..." -ForegroundColor Red
        docker-compose down
        Write-Host "✅ All services stopped!" -ForegroundColor Green
    }
    
    "build" {
        Write-Host "🔨 Building ExplainIQ services..." -ForegroundColor Yellow
        if ($Service) {
            docker-compose build $Service
        } else {
            docker-compose build
        }
        Write-Host "✅ Build complete!" -ForegroundColor Green
    }
    
    "logs" {
        Write-Host "📋 Showing ExplainIQ logs..." -ForegroundColor Cyan
        if ($Service) {
            docker-compose logs -f $Service
        } else {
            docker-compose logs -f
        }
    }
    
    "status" {
        Write-Host "📊 ExplainIQ service status:" -ForegroundColor Cyan
        docker-compose ps
    }
    
    "restart" {
        Write-Host "🔄 Restarting ExplainIQ services..." -ForegroundColor Yellow
        if ($Service) {
            docker-compose restart $Service
        } else {
            docker-compose restart
        }
        Write-Host "✅ Services restarted!" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "💡 Usage examples:" -ForegroundColor Gray
Write-Host "   .\docker-run.ps1 up          # Start all services" -ForegroundColor Gray
Write-Host "   .\docker-run.ps1 dev         # Start in development mode" -ForegroundColor Gray
Write-Host "   .\docker-run.ps1 down        # Stop all services" -ForegroundColor Gray
Write-Host "   .\docker-run.ps1 logs        # View logs" -ForegroundColor Gray
Write-Host "   .\docker-run.ps1 status      # Check status" -ForegroundColor Gray
