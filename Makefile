# Makefile for ExplainIQ Agent
# Usage: make <target>
# Examples:
#   make frontend    - Start frontend services
#   make backend     - Start backend services
#   make agents      - Start agent services only
#   make full        - Start everything
#   make build-frontend - Build frontend only
#   make build-backend  - Build backend only
#   make down        - Stop all services

.PHONY: help frontend backend agents full build-frontend build-backend build-agents build-full down logs clean dev

# Default target
.DEFAULT_GOAL := help

# Docker Compose file location
COMPOSE_FILE := docker/docker-compose.yml
COMPOSE_DIR := docker

# Help target
help:
	@echo "ExplainIQ Agent - Docker Compose Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  make frontend         - Start frontend services (frontend + orchestrator)"
	@echo "  make backend          - Start backend services (orchestrator + all agents)"
	@echo "  make agents           - Start agent services only"
	@echo "  make full             - Start everything (full stack)"
	@echo ""
	@echo "Build targets:"
	@echo "  make build-frontend   - Build frontend services only"
	@echo "  make build-backend    - Build backend services only"
	@echo "  make build-agents     - Build agent services only"
	@echo "  make build-full       - Build everything"
	@echo ""
	@echo "Management targets:"
	@echo "  make down             - Stop all services"
	@echo "  make down-frontend    - Stop frontend services"
	@echo "  make down-backend     - Stop backend services"
	@echo "  make down-agents      - Stop agent services"
	@echo "  make logs             - View logs for all services"
	@echo "  make logs-frontend    - View logs for frontend"
	@echo "  make logs-backend     - View logs for backend"
	@echo "  make clean            - Remove containers, networks, and volumes"
	@echo "  make dev              - Start in development mode (hot-reload)"
	@echo ""
	@echo "Google Cloud Deployment targets:"
	@echo "  make gcloud-help              - Show Google Cloud deployment help"
	@echo "  make gcloud-deploy            - Full deployment (recommended - does everything)"
	@echo "  make gcloud-setup             - Setup GCP project and APIs"
	@echo "  make gcloud-check             - Check GCP configuration"
	@echo "  make gcloud-status            - Check service status"
	@echo "  make gcloud-urls              - Get service URLs"
	@echo "  make gcloud-logs              - View service logs"
	@echo ""
	@echo "  See 'make gcloud-help' for all deployment options"
	@echo ""
	@echo "Service-specific targets:"
	@echo "  make build-orchestrator   - Build orchestrator only"
	@echo "  make build-agent-summarizer - Build agent-summarizer only"
	@echo "  make build-agent-explainer  - Build agent-explainer only"
	@echo "  make build-agent-critic   - Build agent-critic only"
	@echo "  make build-agent-visualizer - Build agent-visualizer only"
	@echo "  make build-frontend-nextjs - Build frontend-nextjs only"

# Start services
frontend:
	@echo "Starting frontend services..."
	cd $(COMPOSE_DIR) && docker-compose --profile frontend up --build

backend:
	@echo "Starting backend services..."
	cd $(COMPOSE_DIR) && docker-compose --profile backend up --build

agents:
	@echo "Starting agent services..."
	cd $(COMPOSE_DIR) && docker-compose --profile agents up --build

full:
	@echo "Starting full stack..."
	cd $(COMPOSE_DIR) && docker-compose --profile full up --build

# Build services (without starting)
build-frontend:
	@echo "Building frontend services..."
	cd $(COMPOSE_DIR) && docker-compose --profile frontend build

build-backend:
	@echo "Building backend services..."
	cd $(COMPOSE_DIR) && docker-compose --profile backend build

build-agents:
	@echo "Building agent services..."
	cd $(COMPOSE_DIR) && docker-compose --profile agents build

build-full:
	@echo "Building full stack..."
	cd $(COMPOSE_DIR) && docker-compose --profile full build

# Build specific services
build-orchestrator:
	@echo "Building orchestrator..."
	cd $(COMPOSE_DIR) && docker-compose build orchestrator

build-agent-summarizer:
	@echo "Building agent-summarizer..."
	cd $(COMPOSE_DIR) && docker-compose build agent-summarizer

build-agent-explainer:
	@echo "Building agent-explainer..."
	cd $(COMPOSE_DIR) && docker-compose build agent-explainer

build-agent-critic:
	@echo "Building agent-critic..."
	cd $(COMPOSE_DIR) && docker-compose build agent-critic

build-agent-visualizer:
	@echo "Building agent-visualizer..."
	cd $(COMPOSE_DIR) && docker-compose build agent-visualizer

build-frontend-nextjs:
	@echo "Building frontend-nextjs..."
	cd $(COMPOSE_DIR) && docker-compose build frontend-nextjs

# Stop services
down:
	@echo "Stopping all services..."
	cd $(COMPOSE_DIR) && docker-compose --profile full down

down-frontend:
	@echo "Stopping frontend services..."
	cd $(COMPOSE_DIR) && docker-compose --profile frontend down

down-backend:
	@echo "Stopping backend services..."
	cd $(COMPOSE_DIR) && docker-compose --profile backend down

down-agents:
	@echo "Stopping agent services..."
	cd $(COMPOSE_DIR) && docker-compose --profile agents down

# View logs
logs:
	@echo "Viewing logs for all services..."
	cd $(COMPOSE_DIR) && docker-compose --profile full logs -f

logs-frontend:
	@echo "Viewing logs for frontend..."
	cd $(COMPOSE_DIR) && docker-compose --profile frontend logs -f

logs-backend:
	@echo "Viewing logs for backend..."
	cd $(COMPOSE_DIR) && docker-compose --profile backend logs -f

logs-orchestrator:
	@echo "Viewing logs for orchestrator..."
	cd $(COMPOSE_DIR) && docker-compose logs -f orchestrator

logs-frontend-nextjs:
	@echo "Viewing logs for frontend-nextjs..."
	cd $(COMPOSE_DIR) && docker-compose logs -f frontend-nextjs

# Development mode
dev:
	@echo "Starting in development mode (hot-reload)..."
	cd $(COMPOSE_DIR) && docker-compose -f docker-compose.dev.yml up

# Clean up
clean:
	@echo "Cleaning up containers, networks, and volumes..."
	cd $(COMPOSE_DIR) && docker-compose down -v --remove-orphans
	@echo "Removing unused Docker images..."
	docker image prune -f

# Rebuild without cache
rebuild-frontend:
	@echo "Rebuilding frontend services (no cache)..."
	cd $(COMPOSE_DIR) && docker-compose --profile frontend build --no-cache

rebuild-backend:
	@echo "Rebuilding backend services (no cache)..."
	cd $(COMPOSE_DIR) && docker-compose --profile backend build --no-cache

rebuild-full:
	@echo "Rebuilding full stack (no cache)..."
	cd $(COMPOSE_DIR) && docker-compose --profile full build --no-cache

# Status check
status:
	@echo "Service status:"
	cd $(COMPOSE_DIR) && docker-compose --profile full ps

# Restart services
restart-frontend:
	@echo "Restarting frontend services..."
	cd $(COMPOSE_DIR) && docker-compose --profile frontend restart

restart-backend:
	@echo "Restarting backend services..."
	cd $(COMPOSE_DIR) && docker-compose --profile backend restart

restart-full:
	@echo "Restarting all services..."
	cd $(COMPOSE_DIR) && docker-compose --profile full restart

# Health check
health:
	@echo "Checking service health..."
	@echo "Orchestrator:"
	@curl -s http://localhost:8080/health || echo "  Not responding"
	@echo "Frontend:"
	@curl -s http://localhost:3000 > /dev/null && echo "  OK" || echo "  Not responding"
	@echo "Agent Summarizer:"
	@curl -s http://localhost:8081/healthz > /dev/null && echo "  OK" || echo "  Not responding"
	@echo "Agent Explainer:"
	@curl -s http://localhost:8082/healthz > /dev/null && echo "  OK" || echo "  Not responding"
	@echo "Agent Critic:"
	@curl -s http://localhost:8083/healthz > /dev/null && echo "  OK" || echo "  Not responding"
	@echo "Agent Visualizer:"
	@curl -s http://localhost:8084/healthz > /dev/null && echo "  OK" || echo "  Not responding"

# Google Cloud Run Deployment
# Set PROJECT_ID environment variable or pass as argument
# Example: make gcloud-build PROJECT_ID=explainiq-production
PROJECT_ID ?= $(shell gcloud config get-value project 2>/dev/null)
REGION ?= europe-west1
REPO ?= explainiq-repo
IMAGE_BASE = $(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)

# Detect OS for script execution
# Use PowerShell script for Windows, bash script for Unix
ifeq ($(OS),Windows_NT)
    UNAME_S := Windows
    DEPLOY_SCRIPT = deploy\deploy-cloudrun.ps1
    DEPLOY_CMD = powershell -ExecutionPolicy Bypass -File
else
    UNAME_S := $(shell uname -s 2>/dev/null || echo "Linux")
    DEPLOY_SCRIPT = deploy/deploy-cloudrun.sh
    DEPLOY_CMD = bash
endif

.PHONY: gcloud-build gcloud-push gcloud-deploy gcloud-build-push gcloud-build-push-deploy \
	gcloud-build-orchestrator gcloud-push-orchestrator gcloud-deploy-orchestrator \
	gcloud-build-summarizer gcloud-push-summarizer gcloud-deploy-summarizer \
	gcloud-build-explainer gcloud-push-explainer gcloud-deploy-explainer \
	gcloud-build-critic gcloud-push-critic gcloud-deploy-critic \
	gcloud-build-visualizer gcloud-push-visualizer gcloud-deploy-visualizer \
	gcloud-build-frontend gcloud-push-frontend gcloud-deploy-frontend \
	gcloud-setup gcloud-check gcloud-help gcloud-deploy-all gcloud-deploy-skip-build \
	gcloud-logs gcloud-logs-orchestrator gcloud-logs-summarizer gcloud-logs-explainer \
	gcloud-logs-critic gcloud-logs-visualizer gcloud-logs-frontend \
	gcloud-status gcloud-urls gcloud-update-env

# Help target for Google Cloud deployment
gcloud-help:
	@echo "Google Cloud Run Build and Deployment"
	@echo ""
	@echo "Configuration:"
	@echo "  PROJECT_ID=$(PROJECT_ID)"
	@echo "  REGION=$(REGION)"
	@echo "  REPO=$(REPO)"
	@echo ""
	@echo "Setup targets:"
	@echo "  make gcloud-setup PROJECT_ID=your-project-id  - Setup GCP project and APIs"
	@echo "  make gcloud-check                              - Check GCP configuration"
	@echo ""
	@echo "Build targets (build Docker images):"
	@echo "  make gcloud-build                              - Build all images"
	@echo "  make gcloud-build-orchestrator                 - Build orchestrator only"
	@echo "  make gcloud-build-summarizer                   - Build summarizer only"
	@echo "  make gcloud-build-explainer                    - Build explainer only"
	@echo "  make gcloud-build-critic                       - Build critic only"
	@echo "  make gcloud-build-visualizer                   - Build visualizer only"
	@echo "  make gcloud-build-frontend                     - Build frontend only"
	@echo ""
	@echo "Push targets (push to Artifact Registry):"
	@echo "  make gcloud-push                               - Push all images"
	@echo "  make gcloud-push-orchestrator                  - Push orchestrator only"
	@echo "  make gcloud-push-summarizer                   - Push summarizer only"
	@echo "  make gcloud-push-explainer                    - Push explainer only"
	@echo "  make gcloud-push-critic                       - Push critic only"
	@echo "  make gcloud-push-visualizer                   - Push visualizer only"
	@echo "  make gcloud-push-frontend                     - Push frontend only"
	@echo ""
	@echo "Deploy targets (deploy to Cloud Run):"
	@echo "  make gcloud-deploy                            - Deploy all services"
	@echo "  make gcloud-deploy-orchestrator               - Deploy orchestrator only"
	@echo "  make gcloud-deploy-summarizer                 - Deploy summarizer only"
	@echo "  make gcloud-deploy-explainer                  - Deploy explainer only"
	@echo "  make gcloud-deploy-critic                     - Deploy critic only"
	@echo "  make gcloud-deploy-visualizer                 - Deploy visualizer only"
	@echo "  make gcloud-deploy-frontend                   - Deploy frontend only"
	@echo ""
	@echo "Combined targets:"
	@echo "  make gcloud-build-push                        - Build and push all images"
	@echo "  make gcloud-build-push-deploy                - Build, push, and deploy all"
	@echo "  make gcloud-deploy                            - Full deployment (recommended)"
	@echo "  make gcloud-deploy-all                       - Full deployment (uses deploy script)"
	@echo "  make gcloud-deploy-skip-build                - Deploy without building"
	@echo ""
	@echo "Management targets:"
	@echo "  make gcloud-logs                             - View logs for all services"
	@echo "  make gcloud-logs-orchestrator                - View orchestrator logs"
	@echo "  make gcloud-status                           - Check service status"
	@echo "  make gcloud-urls                             - Get service URLs"
	@echo "  make gcloud-update-env                       - Update environment variables"
	@echo ""
	@echo "Examples:"
	@echo "  make gcloud-deploy PROJECT_ID=my-project"
	@echo "  make gcloud-build-push-deploy PROJECT_ID=my-project"
	@echo "  make gcloud-build-orchestrator PROJECT_ID=my-project REGION=us-east1"
	@echo "  make gcloud-logs PROJECT_ID=my-project"

# Check GCP configuration
gcloud-check:
ifeq ($(PROJECT_ID),)
	@echo Error: PROJECT_ID is not set
	@echo Set it with: make gcloud-* PROJECT_ID=your-project-id
	@echo Or run: gcloud config set project YOUR_PROJECT_ID
	@exit 1
endif
	@echo "GCP Configuration:"
	@echo "  Project ID: $(PROJECT_ID)"
	@echo "  Region: $(REGION)"
	@echo "  Repository: $(REPO)"
	@echo "  Image Base: $(IMAGE_BASE)"
	@echo ""
	@echo "Checking gcloud authentication..."
ifeq ($(OS),Windows_NT)
	@gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>nul | findstr /R "." >nul || (echo Error: Not authenticated. Run: gcloud auth login && exit 1)
else
	@gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null | head -1 || (echo "Error: Not authenticated. Run: gcloud auth login" && exit 1)
endif
	@echo "✓ Authenticated"
	@echo ""
	@echo "Checking project access..."
ifeq ($(OS),Windows_NT)
	@gcloud projects describe $(PROJECT_ID) --format="value(projectId)" 2>nul || (echo. && echo Error: Cannot access project $(PROJECT_ID) && echo. && echo Possible issues: && echo   1. Project does not exist && echo   2. Project ID is incorrect && echo   3. You don't have permissions && echo   4. Not authenticated to the correct account && echo. && echo Try: && echo   gcloud projects list && echo   gcloud config set project YOUR_PROJECT_ID && echo   gcloud auth login && exit 1)
else
	@gcloud projects describe $(PROJECT_ID) --format="value(projectId)" 2>/dev/null || (echo "" && echo "Error: Cannot access project $(PROJECT_ID)" && echo "" && echo "Possible issues:" && echo "  1. Project does not exist" && echo "  2. Project ID is incorrect" && echo "  3. You don't have permissions" && echo "  4. Not authenticated to the correct account" && echo "" && echo "Try:" && echo "  gcloud projects list" && echo "  gcloud config set project YOUR_PROJECT_ID" && echo "  gcloud auth login" && exit 1)
endif
	@echo "✓ Project accessible"

# Setup GCP project and APIs
gcloud-setup: gcloud-check
	@echo "Setting up Google Cloud project..."
	@gcloud config set project $(PROJECT_ID)
	@echo "Enabling required APIs..."
	@gcloud services enable \
		run.googleapis.com \
		cloudbuild.googleapis.com \
		artifactregistry.googleapis.com \
		secretmanager.googleapis.com \
		storage.googleapis.com \
		--quiet
	@echo "Creating Artifact Registry repository..."
ifeq ($(OS),Windows_NT)
	@gcloud artifacts repositories describe $(REPO) --location=$(REGION) >nul 2>nul || \
	gcloud artifacts repositories create $(REPO) \
		--repository-format=docker \
		--location=$(REGION) \
		--description="ExplainIQ Docker images" \
		--quiet >nul 2>nul || echo Repository already exists or created
else
	@gcloud artifacts repositories describe $(REPO) --location=$(REGION) >/dev/null 2>/dev/null || \
	gcloud artifacts repositories create $(REPO) \
		--repository-format=docker \
		--location=$(REGION) \
		--description="ExplainIQ Docker images" \
		--quiet >/dev/null 2>/dev/null || echo "Repository already exists or created"
endif
	@echo "Configuring Docker authentication..."
ifeq ($(OS),Windows_NT)
	@gcloud auth configure-docker $(REGION)-docker.pkg.dev --quiet >nul 2>nul || echo Docker auth configured
else
	@gcloud auth configure-docker $(REGION)-docker.pkg.dev --quiet >/dev/null 2>/dev/null || echo "Docker auth configured"
endif
	@echo "✓ Setup complete"

# Build Docker images
gcloud-build: gcloud-check
	@echo "Building Docker images for Google Cloud..."
	@echo "Building orchestrator..."
	@docker build -f docker/Dockerfile.orchestrator -t $(IMAGE_BASE)/explainiq-orchestrator:latest .
	@echo "Building agent-summarizer..."
	@docker build -f docker/Dockerfile.agent-summarizer -t $(IMAGE_BASE)/explainiq-summarizer:latest .
	@echo "Building agent-explainer..."
	@docker build -f docker/Dockerfile.agent-explainer -t $(IMAGE_BASE)/explainiq-explainer:latest .
	@echo "Building agent-critic..."
	@docker build -f docker/Dockerfile.agent-critic -t $(IMAGE_BASE)/explainiq-critic:latest .
	@echo "Building agent-visualizer..."
	@docker build -f docker/Dockerfile.agent-visualizer -t $(IMAGE_BASE)/explainiq-visualizer:latest .
	@echo "Building frontend..."
	@docker build -f cmd/frontend/nextjs/Dockerfile -t $(IMAGE_BASE)/explainiq-frontend:latest cmd/frontend/nextjs
	@echo "✓ All images built successfully"

# Build individual services
gcloud-build-orchestrator: gcloud-check
	@echo "Building orchestrator..."
	@docker build -f docker/Dockerfile.orchestrator -t $(IMAGE_BASE)/explainiq-orchestrator:latest .

gcloud-build-summarizer: gcloud-check
	@echo "Building agent-summarizer..."
	@docker build -f docker/Dockerfile.agent-summarizer -t $(IMAGE_BASE)/explainiq-summarizer:latest .

gcloud-build-explainer: gcloud-check
	@echo "Building agent-explainer..."
	@docker build -f docker/Dockerfile.agent-explainer -t $(IMAGE_BASE)/explainiq-explainer:latest .

gcloud-build-critic: gcloud-check
	@echo "Building agent-critic..."
	@docker build -f docker/Dockerfile.agent-critic -t $(IMAGE_BASE)/explainiq-critic:latest .

gcloud-build-visualizer: gcloud-check
	@echo "Building agent-visualizer..."
	@docker build -f docker/Dockerfile.agent-visualizer -t $(IMAGE_BASE)/explainiq-visualizer:latest .

gcloud-build-frontend: gcloud-check
	@echo "Building frontend..."
	@docker build -f cmd/frontend/nextjs/Dockerfile -t $(IMAGE_BASE)/explainiq-frontend:latest cmd/frontend/nextjs

# Push images to Artifact Registry
gcloud-push: gcloud-check
	@echo "Pushing Docker images to Artifact Registry..."
	@docker push $(IMAGE_BASE)/explainiq-orchestrator:latest
	@docker push $(IMAGE_BASE)/explainiq-summarizer:latest
	@docker push $(IMAGE_BASE)/explainiq-explainer:latest
	@docker push $(IMAGE_BASE)/explainiq-critic:latest
	@docker push $(IMAGE_BASE)/explainiq-visualizer:latest
	@docker push $(IMAGE_BASE)/explainiq-frontend:latest
	@echo "✓ All images pushed successfully"

# Push individual services
gcloud-push-orchestrator: gcloud-check
	@docker push $(IMAGE_BASE)/explainiq-orchestrator:latest

gcloud-push-summarizer: gcloud-check
	@docker push $(IMAGE_BASE)/explainiq-summarizer:latest

gcloud-push-explainer: gcloud-check
	@docker push $(IMAGE_BASE)/explainiq-explainer:latest

gcloud-push-critic: gcloud-check
	@docker push $(IMAGE_BASE)/explainiq-critic:latest

gcloud-push-visualizer: gcloud-check
	@docker push $(IMAGE_BASE)/explainiq-visualizer:latest

gcloud-push-frontend: gcloud-check
	@docker push $(IMAGE_BASE)/explainiq-frontend:latest

# Build and push
gcloud-build-push: gcloud-build gcloud-push
	@echo "✓ Build and push complete"

# Deploy to Cloud Run (using deployment script)
gcloud-deploy-all: gcloud-check
	@echo "Deploying all services to Cloud Run..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File $(DEPLOY_SCRIPT) -ProjectId $(PROJECT_ID) -Region $(REGION) -Repo $(REPO)
else
	@chmod +x $(DEPLOY_SCRIPT) && \
	$(DEPLOY_CMD) $(DEPLOY_SCRIPT) --project-id $(PROJECT_ID) --region $(REGION) --repo $(REPO)
endif

gcloud-deploy-skip-build: gcloud-check
	@echo "Deploying to Cloud Run (skipping build)..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File $(DEPLOY_SCRIPT) -ProjectId $(PROJECT_ID) -Region $(REGION) -Repo $(REPO) -SkipBuild
else
	@chmod +x $(DEPLOY_SCRIPT) && \
	$(DEPLOY_CMD) $(DEPLOY_SCRIPT) --project-id $(PROJECT_ID) --region $(REGION) --repo $(REPO) --skip-build
endif

# Deploy individual services (manual deployment)
# Note: Individual service deployment is recommended via gcloud-deploy-all
# These targets are for advanced use cases
gcloud-deploy-orchestrator: gcloud-check
	@echo "Deploying orchestrator to Cloud Run..."
	@echo "Note: Use 'make gcloud-deploy-all' for full deployment with proper configuration"
	@TMP_FILE=$$(mktemp 2>/dev/null || echo /tmp/orchestrator.yaml); \
	sed "s|REGION-docker.pkg.dev/PROJECT_ID/REPO|$(IMAGE_BASE)|g; s|PROJECT_ID|$(PROJECT_ID)|g; s|REGION|$(REGION)|g; s|REPO|$(REPO)|g" deploy/cloudrun/orchestrator.yaml > $$TMP_FILE; \
	gcloud run services replace $$TMP_FILE --region=$(REGION) --platform=managed --quiet; \
	rm -f $$TMP_FILE
	@echo "✓ Orchestrator deployed"

gcloud-deploy-summarizer: gcloud-check
	@echo "Deploying summarizer to Cloud Run..."
	@TMP_FILE=$$(mktemp 2>/dev/null || echo /tmp/summarizer.yaml); \
	sed "s|REGION-docker.pkg.dev/PROJECT_ID/REPO|$(IMAGE_BASE)|g; s|PROJECT_ID|$(PROJECT_ID)|g; s|REGION|$(REGION)|g; s|REPO|$(REPO)|g" deploy/cloudrun/summarizer.yaml > $$TMP_FILE; \
	gcloud run services replace $$TMP_FILE --region=$(REGION) --platform=managed --no-allow-unauthenticated --quiet; \
	rm -f $$TMP_FILE
	@echo "✓ Summarizer deployed"

gcloud-deploy-explainer: gcloud-check
	@echo "Deploying explainer to Cloud Run..."
	@TMP_FILE=$$(mktemp 2>/dev/null || echo /tmp/explainer.yaml); \
	sed "s|REGION-docker.pkg.dev/PROJECT_ID/REPO|$(IMAGE_BASE)|g; s|PROJECT_ID|$(PROJECT_ID)|g; s|REGION|$(REGION)|g; s|REPO|$(REPO)|g" deploy/cloudrun/explainer.yaml > $$TMP_FILE; \
	gcloud run services replace $$TMP_FILE --region=$(REGION) --platform=managed --no-allow-unauthenticated --quiet; \
	rm -f $$TMP_FILE
	@echo "✓ Explainer deployed"

gcloud-deploy-critic: gcloud-check
	@echo "Deploying critic to Cloud Run..."
	@TMP_FILE=$$(mktemp 2>/dev/null || echo /tmp/critic.yaml); \
	sed "s|REGION-docker.pkg.dev/PROJECT_ID/REPO|$(IMAGE_BASE)|g; s|PROJECT_ID|$(PROJECT_ID)|g; s|REGION|$(REGION)|g; s|REPO|$(REPO)|g" deploy/cloudrun/critic.yaml > $$TMP_FILE; \
	gcloud run services replace $$TMP_FILE --region=$(REGION) --platform=managed --no-allow-unauthenticated --quiet; \
	rm -f $$TMP_FILE
	@echo "✓ Critic deployed"

gcloud-deploy-visualizer: gcloud-check
	@echo "Deploying visualizer to Cloud Run..."
	@TMP_FILE=$$(mktemp 2>/dev/null || echo /tmp/visualizer.yaml); \
	sed "s|REGION-docker.pkg.dev/PROJECT_ID/REPO|$(IMAGE_BASE)|g; s|PROJECT_ID|$(PROJECT_ID)|g; s|REGION|$(REGION)|g; s|REPO|$(REPO)|g" deploy/cloudrun/visualizer.yaml > $$TMP_FILE; \
	gcloud run services replace $$TMP_FILE --region=$(REGION) --platform=managed --no-allow-unauthenticated --quiet; \
	rm -f $$TMP_FILE
	@echo "✓ Visualizer deployed"

gcloud-deploy-frontend: gcloud-check
	@echo "Deploying frontend to Cloud Run..."
	@TMP_FILE=$$(mktemp 2>/dev/null || echo /tmp/frontend.yaml); \
	sed "s|REGION-docker.pkg.dev/PROJECT_ID/REPO|$(IMAGE_BASE)|g; s|PROJECT_ID|$(PROJECT_ID)|g; s|REGION|$(REGION)|g; s|REPO|$(REPO)|g" deploy/cloudrun/frontend.yaml > $$TMP_FILE; \
	gcloud run services replace $$TMP_FILE --region=$(REGION) --platform=managed --quiet; \
	rm -f $$TMP_FILE
	@echo "✓ Frontend deployed"

# Full workflow: build, push, and deploy
gcloud-build-push-deploy: gcloud-build-push gcloud-deploy-all
	@echo "✓ Full deployment complete!"

# Main deploy target (recommended - does everything)
gcloud-deploy: gcloud-setup gcloud-build-push-deploy
	@echo ""
	@echo "✓ Deployment complete! Use 'make gcloud-urls' to get service URLs"

# View logs
gcloud-logs: gcloud-check
	@echo "Viewing logs for all services..."
	@gcloud run services logs read explainiq-orchestrator --region=$(REGION) --limit=50
	@gcloud run services logs read explainiq-summarizer --region=$(REGION) --limit=50
	@gcloud run services logs read explainiq-explainer --region=$(REGION) --limit=50
	@gcloud run services logs read explainiq-critic --region=$(REGION) --limit=50
	@gcloud run services logs read explainiq-visualizer --region=$(REGION) --limit=50
	@gcloud run services logs read explainiq-frontend --region=$(REGION) --limit=50

gcloud-logs-orchestrator: gcloud-check
	@gcloud run services logs read explainiq-orchestrator --region=$(REGION) --limit=100 --follow

gcloud-logs-summarizer: gcloud-check
	@gcloud run services logs read explainiq-summarizer --region=$(REGION) --limit=100 --follow

gcloud-logs-explainer: gcloud-check
	@gcloud run services logs read explainiq-explainer --region=$(REGION) --limit=100 --follow

gcloud-logs-critic: gcloud-check
	@gcloud run services logs read explainiq-critic --region=$(REGION) --limit=100 --follow

gcloud-logs-visualizer: gcloud-check
	@gcloud run services logs read explainiq-visualizer --region=$(REGION) --limit=100 --follow

gcloud-logs-frontend: gcloud-check
	@gcloud run services logs read explainiq-frontend --region=$(REGION) --limit=100 --follow

# Check service status
gcloud-status: gcloud-check
	@echo "Cloud Run Service Status:"
	@echo ""
	@gcloud run services list --region=$(REGION) --format="table(metadata.name,status.url,status.conditions[0].status,spec.template.spec.containers[0].image)" --filter="metadata.name:explainiq-*"

# Get service URLs
gcloud-urls: gcloud-check
	@echo "Service URLs:"
	@echo ""
	@echo "Orchestrator:"
	@gcloud run services describe explainiq-orchestrator --region=$(REGION) --format="value(status.url)" || echo "  Not deployed"
	@echo ""
	@echo "Frontend:"
	@gcloud run services describe explainiq-frontend --region=$(REGION) --format="value(status.url)" || echo "  Not deployed"
	@echo ""
	@echo "Agent Services (internal):"
	@echo "  Summarizer:"
	@gcloud run services describe explainiq-summarizer --region=$(REGION) --format="value(status.url)" || echo "    Not deployed"
	@echo "  Explainer:"
	@gcloud run services describe explainiq-explainer --region=$(REGION) --format="value(status.url)" || echo "    Not deployed"
	@echo "  Critic:"
	@gcloud run services describe explainiq-critic --region=$(REGION) --format="value(status.url)" || echo "    Not deployed"
	@echo "  Visualizer:"
	@gcloud run services describe explainiq-visualizer --region=$(REGION) --format="value(status.url)" || echo "    Not deployed"

# Update environment variables
gcloud-update-env: gcloud-check
	@echo "Updating environment variables..."
	@echo ""
	@echo "Getting service URLs..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "$$ORCHESTRATOR_URL = gcloud run services describe explainiq-orchestrator --region=$(REGION) --format='value(status.url)' 2>$$null; $$SUMMARIZER_URL = gcloud run services describe explainiq-summarizer --region=$(REGION) --format='value(status.url)' 2>$$null; $$EXPLAINER_URL = gcloud run services describe explainiq-explainer --region=$(REGION) --format='value(status.url)' 2>$$null; $$CRITIC_URL = gcloud run services describe explainiq-critic --region=$(REGION) --format='value(status.url)' 2>$$null; $$VISUALIZER_URL = gcloud run services describe explainiq-visualizer --region=$(REGION) --format='value(status.url)' 2>$$null; if ($$ORCHESTRATOR_URL -and $$SUMMARIZER_URL) { gcloud run services update explainiq-orchestrator --region=$(REGION) --update-env-vars='AGENT_SUMMARIZER_URL=' + $$SUMMARIZER_URL + ',AGENT_EXPLAINER_URL=' + $$EXPLAINER_URL + ',AGENT_CRITIC_URL=' + $$CRITIC_URL + ',AGENT_VISUALIZER_URL=' + $$VISUALIZER_URL + ',GCP_PROJECT_ID=$(PROJECT_ID),SERVICE_URL=' + $$ORCHESTRATOR_URL --quiet; gcloud run services update explainiq-frontend --region=$(REGION) --update-env-vars='NEXT_PUBLIC_ORCHESTRATOR_URL=' + $$ORCHESTRATOR_URL + ',GCS_PROJECT_ID=$(PROJECT_ID)' --quiet; Write-Host '✓ Environment variables updated' } else { Write-Host 'Error: Services not deployed. Run make gcloud-deploy first'; exit 1 }"
else
	@ORCHESTRATOR_URL=$$(gcloud run services describe explainiq-orchestrator --region=$(REGION) --format="value(status.url)" 2>/dev/null); \
	SUMMARIZER_URL=$$(gcloud run services describe explainiq-summarizer --region=$(REGION) --format="value(status.url)" 2>/dev/null); \
	EXPLAINER_URL=$$(gcloud run services describe explainiq-explainer --region=$(REGION) --format="value(status.url)" 2>/dev/null); \
	CRITIC_URL=$$(gcloud run services describe explainiq-critic --region=$(REGION) --format="value(status.url)" 2>/dev/null); \
	VISUALIZER_URL=$$(gcloud run services describe explainiq-visualizer --region=$(REGION) --format="value(status.url)" 2>/dev/null); \
	if [ -n "$$ORCHESTRATOR_URL" ] && [ -n "$$SUMMARIZER_URL" ]; then \
		echo "Updating orchestrator environment variables..."; \
		gcloud run services update explainiq-orchestrator \
			--region=$(REGION) \
			--update-env-vars="AGENT_SUMMARIZER_URL=$$SUMMARIZER_URL,AGENT_EXPLAINER_URL=$$EXPLAINER_URL,AGENT_CRITIC_URL=$$CRITIC_URL,AGENT_VISUALIZER_URL=$$VISUALIZER_URL,GCP_PROJECT_ID=$(PROJECT_ID),SERVICE_URL=$$ORCHESTRATOR_URL" \
			--quiet; \
		echo "Updating frontend environment variables..."; \
		gcloud run services update explainiq-frontend \
			--region=$(REGION) \
			--update-env-vars="NEXT_PUBLIC_ORCHESTRATOR_URL=$$ORCHESTRATOR_URL,GCS_PROJECT_ID=$(PROJECT_ID)" \
			--quiet; \
		echo "✓ Environment variables updated"; \
	else \
		echo "Error: Services not deployed. Run 'make gcloud-deploy' first"; \
		exit 1; \
	fi
endif

# Legacy deploy targets (for backward compatibility)
deploy: gcloud-deploy
deploy-skip-build: gcloud-deploy-skip-build
deploy-help: gcloud-help
deploy-check: gcloud-check
