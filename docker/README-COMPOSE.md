# Docker Compose Files Guide

This directory contains multiple Docker Compose files optimized for different use cases. This allows you to build only the services you need, significantly reducing build times.

## Available Compose Files

### 1. `docker-compose.full.yml` - Full Stack
**Use when:** You need all services running
```bash
docker-compose -f docker-compose.full.yml up --build
```

### 2. `docker-compose.base.yml` - Base Services
**Use when:** You only need orchestrator and frontend
```bash
docker-compose -f docker-compose.base.yml up --build
```
**Services:**
- orchestrator (port 8080)
- frontend-nextjs (port 3000)

### 3. `docker-compose.agents.yml` - Agent Services Only
**Use when:** You only need to build/rebuild agent services
```bash
docker-compose -f docker-compose.agents.yml up --build
```
**Services:**
- agent-summarizer (port 8081)
- agent-explainer (port 8082)
- agent-critic (port 8083)
- agent-visualizer (port 8084)

### 4. `docker-compose.orchestrator.yml` - Orchestrator Only
**Use when:** You only need the orchestrator for backend testing
```bash
docker-compose -f docker-compose.orchestrator.yml up --build
```

### 5. `docker-compose.frontend.yml` - Frontend Only
**Use when:** You only need the frontend (requires orchestrator running separately)
```bash
docker-compose -f docker-compose.frontend.yml up --build
```

### 6. `docker-compose.dev.yml` - Development Mode
**Use when:** Developing with hot-reload and volume mounts
```bash
docker-compose -f docker-compose.dev.yml up
```

## Combining Multiple Files

You can combine multiple compose files to build specific service groups:

```bash
# Base services + Agents (same as full.yml)
docker-compose -f docker-compose.base.yml -f docker-compose.agents.yml up --build

# Orchestrator + Specific Agent
docker-compose -f docker-compose.orchestrator.yml -f docker-compose.agents.yml up --build agent-summarizer
```

## Common Workflows

### Development Workflow
```bash
# Start base services (orchestrator + frontend)
docker-compose -f docker-compose.base.yml up

# In another terminal, rebuild only the agent you're working on
docker-compose -f docker-compose.agents.yml up --build agent-explainer
```

### Quick Frontend Changes
```bash
# Rebuild only frontend
docker-compose -f docker-compose.frontend.yml up --build
```

### Rebuild All Agents
```bash
# Rebuild all agent services
docker-compose -f docker-compose.agents.yml build --no-cache
```

### Production Build
```bash
# Build everything for production
docker-compose -f docker-compose.full.yml build --no-cache
```

## Build Optimization Tips

1. **Use BuildKit cache:** All compose files use `BUILDKIT_INLINE_CACHE: 1` for better caching
2. **Build specific services:** Use service names to build only what you need
   ```bash
   docker-compose -f docker-compose.agents.yml build agent-explainer
   ```
3. **No-cache when needed:** Use `--no-cache` flag when you want a fresh build
   ```bash
   docker-compose -f docker-compose.base.yml build --no-cache
   ```
4. **Parallel builds:** Docker Compose builds services in parallel when possible

## Environment Variables

All compose files use the `.env` file in the `docker/` directory. Make sure to:
1. Copy `env.example` to `.env`
2. Fill in required variables (especially `GEMINI_API_KEY`)

## Network

All services use the `explainiq-network` bridge network, so they can communicate with each other regardless of which compose file you use.

## Troubleshooting

### Port Conflicts
If you get port conflicts, you can override ports in your command:
```bash
docker-compose -f docker-compose.base.yml up -p 3001:3000 frontend-nextjs
```

### Build Cache Issues
Clear Docker build cache if builds are failing:
```bash
docker builder prune
```

### Network Issues
If services can't communicate, ensure they're using the same network:
```bash
docker network ls | grep explainiq-network
```

