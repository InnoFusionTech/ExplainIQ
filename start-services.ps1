# ExplainIQ Service Startup Script
# This script starts Docker Desktop (if needed) and then starts all services via docker-compose

Write-Host "ExplainIQ Service Startup Script" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan
Write-Host ""

# Check if Docker Desktop is running
Write-Host "Checking Docker Desktop status..." -ForegroundColor Yellow
$dockerRunning = docker ps 2>&1 | Select-String -Pattern "CONTAINER|error" | Out-Null

if ($LASTEXITCODE -ne 0) {
    Write-Host "Docker Desktop is not running. Attempting to start..." -ForegroundColor Yellow
    
    # Try to start Docker Desktop
    $dockerDesktopPath = "$env:ProgramFiles\Docker\Docker\Docker Desktop.exe"
    if (Test-Path $dockerDesktopPath) {
        Start-Process $dockerDesktopPath
        Write-Host "Docker Desktop is starting. Please wait..." -ForegroundColor Yellow
        Write-Host "Waiting for Docker to be ready (this may take 30-60 seconds)..." -ForegroundColor Yellow
        
        # Wait for Docker to be ready
        $maxWait = 60
        $waited = 0
        while ($waited -lt $maxWait) {
            Start-Sleep -Seconds 2
            $waited += 2
            docker ps 2>&1 | Out-Null
            if ($LASTEXITCODE -eq 0) {
                Write-Host "Docker is ready!" -ForegroundColor Green
                break
            }
            Write-Host "." -NoNewline
        }
        Write-Host ""
        
        if ($waited -ge $maxWait) {
            Write-Host "Docker did not start in time. Please start Docker Desktop manually and try again." -ForegroundColor Red
            exit 1
        }
    } else {
        Write-Host "Docker Desktop not found at: $dockerDesktopPath" -ForegroundColor Red
        Write-Host "Please install Docker Desktop or start it manually." -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "Docker is running!" -ForegroundColor Green
}

Write-Host ""
Write-Host "Checking environment file..." -ForegroundColor Yellow
$envFile = "docker\.env"
if (-not (Test-Path $envFile)) {
    Write-Host "Warning: .env file not found. Copying from env.example..." -ForegroundColor Yellow
    if (Test-Path "docker\env.example") {
        Copy-Item "docker\env.example" $envFile
        Write-Host "Please edit docker\.env and set your GEMINI_API_KEY" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "Building and starting services..." -ForegroundColor Cyan
Write-Host ""

# Navigate to project root
Set-Location (Split-Path -Parent $PSScriptRoot)

# Build and start services
docker-compose -f docker/docker-compose.yml up -d --build

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "Services started successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Service URLs:" -ForegroundColor Cyan
    Write-Host "  - Orchestrator:     http://localhost:8080" -ForegroundColor White
    Write-Host "  - Agent Summarizer: http://localhost:8081" -ForegroundColor White
    Write-Host "  - Agent Explainer:  http://localhost:8082" -ForegroundColor White
    Write-Host "  - Agent Critic:     http://localhost:8083" -ForegroundColor White
    Write-Host "  - Agent Visualizer: http://localhost:8084" -ForegroundColor White
    Write-Host "  - Frontend (NextJS): http://localhost:3000" -ForegroundColor White
    Write-Host ""
    Write-Host "View logs: docker-compose -f docker/docker-compose.yml logs -f" -ForegroundColor Yellow
    Write-Host "Stop services: docker-compose -f docker/docker-compose.yml down" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Opening frontend in browser..." -ForegroundColor Cyan
    Start-Sleep -Seconds 3
    Start-Process "http://localhost:3000"
} else {
    Write-Host ""
    Write-Host "Failed to start services. Check the error messages above." -ForegroundColor Red
    exit 1
}










