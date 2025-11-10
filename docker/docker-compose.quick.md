# Quick Docker Compose Commands

## Simple Commands

### Frontend Only
**PowerShell (Windows):**
```powershell
cd docker
.\docker-compose.ps1 -Frontend up --build
```

**Bash/Linux/Mac:**
```bash
cd docker
docker-compose --profile frontend up --build
```

### Backend Only (Orchestrator + All Agents)
**PowerShell (Windows):**
```powershell
cd docker
.\docker-compose.ps1 -Backend up --build
```

**Bash/Linux/Mac:**
```bash
cd docker
docker-compose --profile backend up --build
```

### Agents Only
**PowerShell (Windows):**
```powershell
cd docker
.\docker-compose.ps1 -Agents up --build
```

**Bash/Linux/Mac:**
```bash
cd docker
docker-compose --profile agents up --build
```

### Everything (Full Stack)
**PowerShell (Windows):**
```powershell
cd docker
.\docker-compose.ps1 up --build
```

**Bash/Linux/Mac:**
```bash
cd docker
docker-compose up --build
```

## Build Specific Services

### Build only frontend
```bash
docker-compose --profile frontend build frontend-nextjs
```

### Build only orchestrator
```bash
docker-compose --profile backend build orchestrator
```

### Build only one agent
```bash
docker-compose --profile agents build agent-explainer
```

## Stop Services

### Stop frontend
```bash
docker-compose --profile frontend down
```

### Stop backend
```bash
docker-compose --profile backend down
```

### Stop everything
```bash
docker-compose down
```

## Development Mode

For development with hot-reload, use:
```bash
docker-compose -f docker-compose.dev.yml up
```

## Examples

### Start just the frontend (requires orchestrator running separately)
```bash
docker-compose --profile frontend up
```

### Start backend services (orchestrator + all agents)
```bash
docker-compose --profile backend up --build
```

### Rebuild and start everything
```bash
docker-compose up --build
```

### Rebuild only frontend after changes
```bash
docker-compose --profile frontend build frontend-nextjs
docker-compose --profile frontend up frontend-nextjs
```

