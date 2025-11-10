# Docker Compose Setup

This directory contains optimized Docker Compose configurations for the ExplainIQ application.

## Quick Start

### Using Make (Recommended - Cross-platform)
```bash
make frontend    # Start frontend services
make backend     # Start backend services
make agents      # Start agent services only
make full        # Start everything
make help        # Show all available commands
```

### Windows (PowerShell)
```powershell
cd docker
.\docker-compose.ps1 -Frontend up --build    # Frontend only
.\docker-compose.ps1 -Backend up --build     # Backend only
.\docker-compose.ps1 up --build              # Everything
```

### Linux/Mac (Bash)
```bash
cd docker
docker-compose --profile frontend up --build    # Frontend only
docker-compose --profile backend up --build     # Backend only
docker-compose up --build                        # Everything
```

## Profiles

The main `docker-compose.yml` uses profiles to allow selective service startup:

- **`frontend`** - Frontend + Orchestrator (frontend depends on orchestrator)
- **`backend`** - Orchestrator + All Agents
- **`agents`** - All Agent services only
- **`full`** - Everything (default when no profile specified)

## Service Ports

- **Orchestrator**: 8080
- **Agent Summarizer**: 8081
- **Agent Explainer**: 8082
- **Agent Critic**: 8083
- **Agent Visualizer**: 8084
- **Frontend (Next.js)**: 3000

## Environment Setup

1. Copy `env.example` to `.env`:
   ```bash
   cp env.example .env
   ```

2. Edit `.env` and fill in required variables:
   - `GEMINI_API_KEY` (required)
   - Other environment variables as needed

## Common Commands

### Using Make (Easiest)
```bash
# Start services
make frontend         # Frontend only
make backend          # Backend only
make agents           # Agents only
make full             # Everything

# Build services
make build-frontend   # Build frontend only
make build-backend    # Build backend only
make build-full       # Build everything

# Stop services
make down             # Stop all
make down-frontend    # Stop frontend
make down-backend     # Stop backend

# View logs
make logs             # All services
make logs-frontend    # Frontend only
make logs-backend     # Backend only

# Other useful commands
make status           # Check service status
make health           # Health check
make clean            # Clean up everything
make dev              # Development mode
```

### Direct Docker Compose Commands

#### Start Services
```bash
# Frontend only
docker-compose --profile frontend up

# Backend only
docker-compose --profile backend up

# Everything
docker-compose up
```

#### Build Services
```bash
# Build frontend only
docker-compose --profile frontend build

# Build backend only
docker-compose --profile backend build

# Build everything
docker-compose build
```

#### Stop Services
```bash
# Stop all services
docker-compose down

# Stop specific profile
docker-compose --profile frontend down
```

#### View Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f orchestrator
```

## Development Mode

For development with hot-reload, use:
```bash
docker-compose -f docker-compose.dev.yml up
```

## Build Optimization

- All services use `BUILDKIT_INLINE_CACHE: 1` for better caching
- Build only what you need using profiles
- Use `--no-cache` when you need a fresh build:
  ```bash
  docker-compose build --no-cache
  ```

## Troubleshooting

### Port Conflicts
If ports are already in use, you can override them:
```bash
docker-compose up -p 3001:3000 frontend-nextjs
```

### Build Cache Issues
Clear Docker build cache:
```bash
docker builder prune
```

### Network Issues
Ensure services are on the same network:
```bash
docker network ls | grep explainiq-network
```

## Additional Files

- `docker-compose.base.yml` - Base services (orchestrator + frontend)
- `docker-compose.agents.yml` - Agent services only
- `docker-compose.full.yml` - Full stack (all services)
- `docker-compose.dev.yml` - Development mode with hot-reload
- `docker-compose.quick.md` - Quick reference guide
