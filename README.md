# ExplainIQ Agent

A comprehensive AI-powered microservices platform for intelligent explanation and analysis, built with Go and optimized for cloud deployment.

## Architecture

The platform consists of six main services:

- **Orchestrator**: Central coordination service for managing workflows
- **Agent Summarizer**: AI-powered text summarization service
- **Agent Explainer**: AI-powered explanation generation service
- **Agent Critic**: AI-powered critique and analysis service
- **Agent Visualizer**: AI-powered visualization generation service
- **Frontend**: Web interface for user interaction

## Quick Start

### Prerequisites

- Go 1.22 or later
- Docker (for containerization)
- Google Cloud SDK (for deployment)
- Make (for build automation)

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/InnoFusionTech/ExplainIQ.git
   cd ExplainIQ
   ```

2. **Setup development environment**
   ```bash
   make setup
   ```

3. **Copy environment variables**
   ```bash
   cp env.example .env
   # Edit .env with your configuration
   ```

4. **Download dependencies**
   ```bash
   make deps
   ```

5. **Build all services**
   ```bash
   make build
   ```

6. **Run tests**
   ```bash
   make test
   ```

7. **Run linting**
   ```bash
   make lint
   ```

### Local Development

Run all services locally:
```bash
make run-local
```

Run a specific service:
```bash
make run-local-orchestrator
```

Stop all services:
```bash
make stop-local
```

### Service Endpoints

When running locally, services are available at:

- Orchestrator: http://localhost:8080
- Agent Summarizer: http://localhost:8081
- Agent Explainer: http://localhost:8082
- Agent Critic: http://localhost:8083
- Agent Visualizer: http://localhost:8084
- Frontend: http://localhost:8085

Each service provides a health check endpoint at `/healthz`.

## Deployment

### Google Cloud Run

1. **Configure Google Cloud**
   ```bash
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   ```

2. **Update Makefile variables**
   ```bash
   # Edit Makefile and update PROJECT_ID
   PROJECT_ID := your-actual-project-id
   ```

3. **Deploy all services**
   ```bash
   make deploy
   ```

4. **Deploy specific service**
   ```bash
   make deploy-orchestrator
   ```

### Docker

Build Docker images:
```bash
make docker-build
```

Push to registry:
```bash
make docker-push
```

## Project Structure

```
ExplainIQ/
├── cmd/                          # Service entry points
│   ├── orchestrator/
│   ├── agent-critic/
│   ├── agent-explainer/
│   ├── agent-summarizer/
│   ├── agent-visualizer/
│   └── frontend/
├── internal/                     # Shared packages
│   ├── adk/                     # Agent Development Kit
│   ├── llm/                     # LLM client abstractions
│   ├── elastic/                 # Elasticsearch client
│   ├── storage/                 # Storage abstractions
│   ├── auth/                    # Authentication utilities
│   ├── telemetry/               # Observability tools
│   └── apiutils/                # API utilities
├── docker/                       # Docker configurations
│   ├── Dockerfile.*
│   └── docker-compose*.yml
├── docs/                         # Documentation
│   ├── AI_MODEL_SETUP.md
│   ├── AUTHENTICATION.md
│   ├── ENVIRONMENT_SETUP.md
│   └── QUOTA_SYSTEM.md
├── deploy/                       # Deployment configurations
│   └── cloudrun/
├── go.work                       # Go workspace file
├── go.mod                        # Root module definition
├── Makefile                      # Build automation
├── .env                          # Environment variables
└── README.md                     # This file
```

## Development

### Adding a New Service

1. Create service directory in `cmd/`
2. Add `go.mod` with proper module path
3. Implement `main.go` with health endpoint
4. Add to `go.work` file
5. Create Cloud Run YAML in `deploy/cloudrun/`
6. Update Makefile with new service
7. Add to CI workflow

### Code Style

- Follow Go standard formatting (`go fmt`)
- Use `golangci-lint` for linting
- Write tests for new functionality
- Document public APIs

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/llm/...
```

## Monitoring and Observability

The platform includes comprehensive observability features:

- **Health Checks**: Each service exposes `/healthz` endpoint
- **Metrics**: Prometheus-compatible metrics
- **Tracing**: Distributed tracing with Jaeger
- **Logging**: Structured logging with logrus

## Security

- JWT-based authentication
- CORS configuration
- Security headers
- Input validation
- Rate limiting

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run linting and tests
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For questions and support, please open an issue in the GitHub repository.

