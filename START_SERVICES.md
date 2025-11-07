# Starting ExplainIQ Services

## Prerequisites

1. **Docker Desktop must be running**
   - Start Docker Desktop application
   - Wait for Docker to fully start (green icon in system tray)

2. **Environment Variables**
   - Ensure `.env` file exists in `docker/` directory
   - Copy from `docker/env.example` if needed
   - Set your `GEMINI_API_KEY` in the `.env` file

## Starting Services

### Option 1: Docker Compose (Recommended)

```powershell
# Navigate to project root
cd "C:\App\ExplainIQ Agent"

# Start all services
docker-compose -f docker/docker-compose.yml up -d --build

# View logs
docker-compose -f docker/docker-compose.yml logs -f

# Check status
docker-compose -f docker/docker-compose.yml ps

# Stop services
docker-compose -f docker/docker-compose.yml down
```

### Option 2: Individual Services (Local Builds)

If Docker Desktop is not available, you can run services directly:

```powershell
# Set environment variables
$env:GEMINI_API_KEY = "your-api-key-here"
$env:PORT = "8080"

# Run orchestrator (in separate terminal)
.\bin\orchestrator.exe

# Run agent services (in separate terminals)
.\bin\agent-summarizer.exe
.\bin\agent-explainer.exe
.\bin\agent-critic.exe
.\bin\agent-visualizer.exe

# Run frontend (in separate terminal)
.\bin\frontend.exe
```

## Service URLs

Once services are running:

- **Orchestrator**: http://localhost:8080
- **Agent Summarizer**: http://localhost:8081
- **Agent Explainer**: http://localhost:8082
- **Agent Critic**: http://localhost:8083
- **Agent Visualizer**: http://localhost:8084
- **Frontend**: http://localhost:8085
- **Next.js Frontend**: http://localhost:3000

## Health Checks

Test if services are running:

```powershell
# Orchestrator
curl http://localhost:8080/health

# Agent Services
curl http://localhost:8081/healthz
curl http://localhost:8082/healthz
curl http://localhost:8083/healthz
curl http://localhost:8084/healthz
```

## Troubleshooting

### Docker Desktop Not Running
1. Start Docker Desktop application
2. Wait for it to fully initialize
3. Check system tray for Docker icon (should be green)

### Port Already in Use
```powershell
# Find process using port
netstat -ano | findstr :8080

# Kill process (replace PID with actual process ID)
taskkill /PID <PID> /F
```

### Environment Variables Not Loaded
- Ensure `.env` file exists in `docker/` directory
- Check file has correct format (no spaces around `=`)
- Restart Docker containers after changing `.env`

### Build Errors
```powershell
# Clean and rebuild
docker-compose -f docker/docker-compose.yml down
docker-compose -f docker/docker-compose.yml build --no-cache
docker-compose -f docker/docker-compose.yml up -d
```

## Next Steps

1. **Start Docker Desktop** (if not already running)
2. **Check `.env` file** has your GEMINI_API_KEY
3. **Run docker-compose** command above
4. **Open browser** to http://localhost:3000 (Next.js frontend)

## Features Available

- ✅ Professional UI with sidebar menu
- ✅ Multiple explanation types (Standard, Visualization, Simple, Analogies)
- ✅ Error boundaries and fault tolerance
- ✅ Automatic retry mechanisms
- ✅ Real-time progress tracking via SSE
- ✅ Professional loading states and animations





