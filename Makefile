# ExplainIQ Makefile
.PHONY: help build test lint run-local deploy clean deps

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Variables
SERVICES := orchestrator agent-summarizer agent-explainer agent-critic agent-visualizer frontend
BIN_DIR := bin
DOCKER_REGISTRY := gcr.io
PROJECT_ID := your-project-id
REGION := us-central1

# Go variables
GO_VERSION := 1.22
GOLANGCI_LINT_VERSION := v1.55

# Build targets
build: deps ## Build all services
	@echo "Building all services..."
	@mkdir -p $(BIN_DIR)
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		cd cmd/$$service && \
		CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o ../../$(BIN_DIR)/$$service . && \
		cd ../..; \
	done
	@echo "Build complete!"

build-%: deps ## Build specific service (e.g., build-orchestrator)
	@echo "Building $*..."
	@mkdir -p $(BIN_DIR)
	cd cmd/$* && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o ../../$(BIN_DIR)/$* .
	@echo "Build complete for $*!"

# Test targets
test: deps ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete!"

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint targets
lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run --timeout=5m
	@echo "Linting complete!"

lint-fix: ## Run golangci-lint with --fix
	@echo "Running linter with fixes..."
	golangci-lint run --timeout=5m --fix
	@echo "Linting complete!"

# Local development targets
run-local: ## Run all services locally (requires build first)
	@echo "Starting all services locally..."
	@for service in $(SERVICES); do \
		echo "Starting $$service on port 808$$(echo $$service | wc -c | tr -d ' ')"; \
		./$(BIN_DIR)/$$service & \
	done
	@echo "All services started. Use 'make stop-local' to stop them."

run-local-%: build-% ## Run specific service locally (e.g., run-local-orchestrator)
	@echo "Starting $* locally..."
	./$(BIN_DIR)/$*

stop-local: ## Stop all locally running services
	@echo "Stopping all services..."
	@pkill -f "$(BIN_DIR)/" || true
	@echo "All services stopped."

# Docker targets
docker-build: ## Build Docker images for all services
	@echo "Building Docker images..."
	@for service in $(SERVICES); do \
		echo "Building Docker image for $$service..."; \
		docker build -f Dockerfile.$$service -t $(DOCKER_REGISTRY)/$(PROJECT_ID)/explainiq-$$service:latest .; \
	done
	@echo "Docker build complete!"

docker-push: docker-build ## Push Docker images to registry
	@echo "Pushing Docker images..."
	@for service in $(SERVICES); do \
		echo "Pushing $$service..."; \
		docker push $(DOCKER_REGISTRY)/$(PROJECT_ID)/explainiq-$$service:latest; \
	done
	@echo "Docker push complete!"

# Deployment targets
deploy: docker-push ## Deploy all services to Cloud Run
	@echo "Deploying to Cloud Run..."
	@for service in $(SERVICES); do \
		echo "Deploying $$service..."; \
		gcloud run deploy explainiq-$$service \
			--image $(DOCKER_REGISTRY)/$(PROJECT_ID)/explainiq-$$service:latest \
			--platform managed \
			--region $(REGION) \
			--allow-unauthenticated \
			--file deploy/cloudrun/$$service.yaml; \
	done
	@echo "Deployment complete!"

deploy-%: docker-build ## Deploy specific service (e.g., deploy-orchestrator)
	@echo "Deploying $* to Cloud Run..."
	gcloud run deploy explainiq-$* \
		--image $(DOCKER_REGISTRY)/$(PROJECT_ID)/explainiq-$*:latest \
		--platform managed \
		--region $(REGION) \
		--allow-unauthenticated \
		--file deploy/cloudrun/$*.yaml
	@echo "Deployment complete for $*!"

# Utility targets
deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	@echo "Dependencies ready!"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete!"

clean-docker: ## Clean Docker images
	@echo "Cleaning Docker images..."
	@for service in $(SERVICES); do \
		docker rmi $(DOCKER_REGISTRY)/$(PROJECT_ID)/explainiq-$$service:latest || true; \
	done
	@echo "Docker clean complete!"

# Development setup
setup: ## Setup development environment
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	fi
	@if ! command -v gcloud >/dev/null 2>&1; then \
		echo "Please install Google Cloud SDK: https://cloud.google.com/sdk/docs/install"; \
	fi
	@echo "Development environment setup complete!"

# Health check
health: ## Check health of all deployed services
	@echo "Checking service health..."
	@for service in $(SERVICES); do \
		echo "Checking $$service..."; \
		curl -f http://localhost:808$$(echo $$service | wc -c | tr -d ' ')/healthz || echo "$$service is not running"; \
	done

# Format code
fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...
	@echo "Code formatted!"

# Vet code
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...
	@echo "Go vet complete!"

# Security scan
security: ## Run security scan
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi
	@echo "Security scan complete!"

