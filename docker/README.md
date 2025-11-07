# ExplainIQ Docker Setup

This directory contains Docker configurations for running the ExplainIQ platform in containers.

## Prerequisites

1. **Docker Desktop** - Make sure Docker Desktop is installed and running
2. **Environment Variables** - Copy `env.example` to `.env` and configure your API keys

## Quick Start

### 1. Setup Environment

```bash
# Copy environment template
cp env.example .env

# Edit .env file with your API keys
# GEMINI_API_KEY=your-actual-api-key-here
```

### 2. Build and Run

```bash
# Build all services
docker-compose build

# Start all services
docker-compose up -d

# Or use the PowerShell script
.\docker-run.ps1 up
```

### 3. Access Services

- **Frontend (Go)**: http://localhost:8085
- **Frontend (Next.js)**: http://localhost:3000
- **API Orchestrator**: http://localhost:8080
- **Agent Services**: 
  - Summarizer: http://localhost:8081
  - Explainer: http://localhost:8082
  - Critic: http://localhost:8083
  - Visualizer: http://localhost:8084

## Available Commands

### Using Docker Compose

```bash
# Start all services
docker-compose up -d

# Start specific service
docker-compose up -d agent-critic

# View logs
docker-compose logs -f

# View logs for specific service
docker-compose logs -f agent-critic

# Stop all services
docker-compose down

# Rebuild and start
docker-compose up --build -d
```

### Using PowerShell Script

```powershell
# Start all services
.\docker-run.ps1 up

# Start in development mode
.\docker-run.ps1 dev

# Stop all services
.\docker-run.ps1 down

# View logs
.\docker-run.ps1 logs

# Check status
.\docker-run.ps1 status

# Build services
.\docker-run.ps1 build

# Restart services
.\docker-run.ps1 restart
```

## Service Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Frontend      │    │   Orchestrator  │
│   (Next.js)     │    │   (Go)          │    │   (Port 8080)   │
│   Port 3000     │    │   Port 8085     │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Summarizer    │    │   Explainer     │    │   Critic        │
│   Port 8081     │    │   Port 8082     │    │   Port 8083     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   Visualizer    │
                    │   Port 8084     │
                    └─────────────────┘
```

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `GEMINI_API_KEY` | Google Gemini AI API key | Yes |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | No |
| `GIN_MODE` | Gin framework mode (debug, release) | No |

## Troubleshooting

### Docker Desktop Not Running
```
error during connect: Head "http://%2F%2F.%2Fpipe%2FdockerDesktopLinuxEngine/_ping"
```
**Solution**: Start Docker Desktop and wait for it to fully initialize.

### Build Failures
```bash
# Clean build cache
docker system prune -a

# Rebuild without cache
docker-compose build --no-cache
```

### Port Conflicts
If you get port conflicts, modify the port mappings in `docker-compose.yml`:
```yaml
ports:
  - "8080:8080"  # Change first number to available port
```

### Service Health Checks
```bash
# Check service health
docker-compose ps

# View service logs
docker-compose logs service-name
```

## Development

### Hot Reload (Development Mode)
```bash
# Use development compose file
docker-compose -f docker-compose.dev.yml up -d
```

### Debugging
```bash
# Run service in interactive mode
docker-compose run --rm agent-critic sh

# View service logs in real-time
docker-compose logs -f agent-critic
```

## Production Deployment

For production deployment, consider:

1. **Environment Variables**: Use proper secrets management
2. **Resource Limits**: Add memory and CPU limits
3. **Health Checks**: Configure proper health check intervals
4. **Logging**: Set up centralized logging
5. **Monitoring**: Add monitoring and alerting
6. **Security**: Use non-root users and security scanning

## Cleanup

```bash
# Stop and remove containers
docker-compose down

# Remove volumes
docker-compose down -v

# Remove images
docker-compose down --rmi all

# Full cleanup
docker system prune -a
```








