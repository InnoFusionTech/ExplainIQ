# Google Cloud Run Deployment Guide

## Deployment Strategy

### Recommended: Single Project with Separate Services

**Best Practice**: Deploy all services in the **same Google Cloud project** as separate Cloud Run services. This provides:

✅ **Benefits**:
- Easier service-to-service authentication (same project)
- Shared IAM policies and service accounts
- Centralized billing and monitoring
- Simpler networking (internal service URLs)
- Better for Google AI Agents Category challenge (all agents in one project)

❌ **Not Recommended**: Multiple projects would require:
- Cross-project authentication complexity
- More complex IAM setup
- Harder to manage and monitor
- Higher operational overhead

## Architecture

```
Google Cloud Project: explainiq-production
│
├── Cloud Run Services:
│   ├── explainiq-orchestrator (public)
│   ├── explainiq-summarizer (internal)
│   ├── explainiq-explainer (internal)
│   ├── explainiq-critic (internal)
│   ├── explainiq-visualizer (internal)
│   └── explainiq-frontend (public)
│
├── Artifact Registry:
│   └── REGION-docker.pkg.dev/PROJECT_ID/REPO/explainiq-*:latest
│
├── Secret Manager:
│   └── google-ai-api-key
│
└── Cloud Storage:
    └── explainiq-diagrams (for images)
```

## Prerequisites

1. **Google Cloud Project**
   ```bash
   # Create or select a project
   gcloud projects create explainiq-production --name="ExplainIQ Production"
   gcloud config set project explainiq-production
   ```

2. **Enable Required APIs**
   ```bash
   gcloud services enable \
     run.googleapis.com \
     cloudbuild.googleapis.com \
     artifactregistry.googleapis.com \
     secretmanager.googleapis.com \
     storage.googleapis.com
   ```

3. **Set Up Authentication**
   ```bash
   gcloud auth login
   gcloud auth application-default login
   ```

4. **Create Artifact Registry**
   ```bash
   gcloud artifacts repositories create explainiq-repo \
     --repository-format=docker \
     --location=europe-west1 \
     --description="ExplainIQ Docker images"
   ```

5. **Create Secret for API Key**
   ```bash
   echo -n "YOUR_GEMINI_API_KEY" | gcloud secrets create google-ai-api-key \
     --data-file=- \
     --replication-policy="automatic"
   ```

## Deployment Steps

### Option 1: Automated Deployment Script

Use the provided deployment script:

```bash
# For Windows (PowerShell)
.\deploy\deploy-cloudrun.ps1 -ProjectId "explainiq-production"

# For Linux/Mac (Bash)
./deploy/deploy-cloudrun.sh --project-id explainiq-production
```

### Option 2: Manual Deployment

#### Step 1: Build and Push Docker Images

```bash
# Set your project ID
export PROJECT_ID=explainiq-production
export REGION=europe-west1
export REPO=explainiq-repo

# Build and push orchestrator
docker build -f docker/Dockerfile.orchestrator -t gcr.io/$PROJECT_ID/explainiq-orchestrator:latest .
docker push gcr.io/$PROJECT_ID/explainiq-orchestrator:latest

# Build and push agents
docker build -f docker/Dockerfile.agent-summarizer -t gcr.io/$PROJECT_ID/explainiq-summarizer:latest .
docker push gcr.io/$PROJECT_ID/explainiq-summarizer:latest

docker build -f docker/Dockerfile.agent-explainer -t gcr.io/$PROJECT_ID/explainiq-explainer:latest .
docker push gcr.io/$PROJECT_ID/explainiq-explainer:latest

docker build -f docker/Dockerfile.agent-critic -t gcr.io/$PROJECT_ID/explainiq-critic:latest .
docker push gcr.io/$PROJECT_ID/explainiq-critic:latest

docker build -f docker/Dockerfile.agent-visualizer -t gcr.io/$PROJECT_ID/explainiq-visualizer:latest .
docker push gcr.io/$PROJECT_ID/explainiq-visualizer:latest

# Build and push frontend
docker build -f cmd/frontend/nextjs/Dockerfile -t gcr.io/$PROJECT_ID/explainiq-frontend:latest cmd/frontend/nextjs
docker push gcr.io/$PROJECT_ID/explainiq-frontend:latest
```

#### Step 2: Update Cloud Run YAML Files

Replace `PROJECT_ID` in all YAML files:

```bash
# Update all YAML files
sed -i 's/PROJECT_ID/explainiq-production/g' deploy/cloudrun/*.yaml
```

#### Step 3: Deploy Services

```bash
# Deploy orchestrator (public)
gcloud run services replace deploy/cloudrun/orchestrator.yaml \
  --region=$REGION \
  --platform=managed

# Deploy agents (internal - no public ingress)
gcloud run services replace deploy/cloudrun/summarizer.yaml \
  --region=$REGION \
  --platform=managed \
  --no-allow-unauthenticated

gcloud run services replace deploy/cloudrun/explainer.yaml \
  --region=$REGION \
  --platform=managed \
  --no-allow-unauthenticated

gcloud run services replace deploy/cloudrun/critic.yaml \
  --region=$REGION \
  --platform=managed \
  --no-allow-unauthenticated

gcloud run services replace deploy/cloudrun/visualizer.yaml \
  --region=$REGION \
  --platform=managed \
  --no-allow-unauthenticated

# Deploy frontend (public)
gcloud run services replace deploy/cloudrun/frontend.yaml \
  --region=$REGION \
  --platform=managed
```

#### Step 4: Configure Service-to-Service Authentication

```bash
# Grant orchestrator permission to invoke agents
gcloud run services add-iam-policy-binding explainiq-summarizer \
  --region=$REGION \
  --member="serviceAccount:explainiq-orchestrator@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.invoker"

gcloud run services add-iam-policy-binding explainiq-explainer \
  --region=$REGION \
  --member="serviceAccount:explainiq-orchestrator@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.invoker"

gcloud run services add-iam-policy-binding explainiq-critic \
  --region=$REGION \
  --member="serviceAccount:explainiq-orchestrator@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.invoker"

gcloud run services add-iam-policy-binding explainiq-visualizer \
  --region=$REGION \
  --member="serviceAccount:explainiq-orchestrator@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.invoker"
```

#### Step 5: Update Orchestrator Environment Variables

```bash
# Get service URLs
ORCHESTRATOR_URL=$(gcloud run services describe explainiq-orchestrator \
  --region=$REGION \
  --format="value(status.url)")

SUMMARIZER_URL=$(gcloud run services describe explainiq-summarizer \
  --region=$REGION \
  --format="value(status.url)")

EXPLAINER_URL=$(gcloud run services describe explainiq-explainer \
  --region=$REGION \
  --format="value(status.url)")

CRITIC_URL=$(gcloud run services describe explainiq-critic \
  --region=$REGION \
  --format="value(status.url)")

VISUALIZER_URL=$(gcloud run services describe explainiq-visualizer \
  --region=$REGION \
  --format="value(status.url)")

# Update orchestrator with agent URLs
gcloud run services update explainiq-orchestrator \
  --region=$REGION \
  --update-env-vars="AGENT_SUMMARIZER_URL=$SUMMARIZER_URL,AGENT_EXPLAINER_URL=$EXPLAINER_URL,AGENT_CRITIC_URL=$CRITIC_URL,AGENT_VISUALIZER_URL=$VISUALIZER_URL"
```

## Environment Variables

### Orchestrator
- `AGENT_SUMMARIZER_URL`: URL of summarizer service
- `AGENT_EXPLAINER_URL`: URL of explainer service
- `AGENT_CRITIC_URL`: URL of critic service
- `AGENT_VISUALIZER_URL`: URL of visualizer service
- `GCP_PROJECT_ID`: Your GCP project ID
- `SERVICE_URL`: Orchestrator's own URL (for auth)

### Agents
- `GEMINI_API_KEY`: From Secret Manager
- `GCP_PROJECT_ID`: Your GCP project ID
- `PORT`: Service port (8081-8084)

### Frontend
- `NEXT_PUBLIC_ORCHESTRATOR_URL`: Orchestrator's public URL
- `GCS_BUCKET`: Google Cloud Storage bucket for images
- `GCS_PROJECT_ID`: Your GCP project ID

## Service URLs

After deployment, get service URLs:

```bash
# Get all service URLs
gcloud run services list --region=europe-west1 --format="table(metadata.name,status.url)"
```

## Networking

### Internal Services (Agents)
- Agents should **NOT** have public ingress
- Only accessible from within the same project
- Use service-to-service authentication

### Public Services
- Orchestrator: Public (needs to be accessible from frontend)
- Frontend: Public (user-facing)

## Cost Optimization

1. **Min Instances**: Set to 0 for agents (scale to zero when not in use)
2. **Max Instances**: Limit to 10 per service
3. **CPU Throttling**: Disabled for better performance
4. **Concurrency**: Tuned per service (50-100)

## Monitoring

1. **Cloud Logging**: All services log to Cloud Logging
2. **Cloud Monitoring**: Set up dashboards for:
   - Request latency
   - Error rates
   - Instance counts
   - Cost tracking

## Security Best Practices

1. **Service Accounts**: Use dedicated service accounts per service
2. **Secrets**: Store API keys in Secret Manager
3. **IAM**: Follow principle of least privilege
4. **VPC**: Consider VPC connector for private networking
5. **CORS**: Configure CORS for frontend-orchestrator communication

## Troubleshooting

### Service Not Starting
```bash
# Check logs
gcloud run services logs read explainiq-orchestrator --region=europe-west1

# Check service status
gcloud run services describe explainiq-orchestrator --region=europe-west1
```

### Authentication Issues
```bash
# Verify service account permissions
gcloud projects get-iam-policy explainiq-production
```

### Image Pull Errors
```bash
# Verify image exists
gcloud container images list --repository=gcr.io/explainiq-production
```

## Next Steps

1. Set up CI/CD pipeline (Cloud Build)
2. Configure monitoring and alerting
3. Set up custom domain
4. Enable Cloud CDN for frontend
5. Configure backup and disaster recovery

