# Manual Cloud Run Deployment (Without Scripts)

## Prerequisites

1. **Secret Manager Secret** (REQUIRED):
   ```bash
   echo -n "YOUR_GEMINI_API_KEY" | gcloud secrets create google-ai-api-key \
     --data-file=- \
     --replication-policy="automatic"
   ```

2. **Cloud Storage Bucket** (REQUIRED):
   ```bash
   gsutil mb -p explainiq-477714 -l europe-west1 gs://explainiq-diagrams
   ```

## Set Variables

```powershell
$PROJECT_ID = "explainiq-477714"
$REGION = "europe-west1"
$REPO = "explainiq-repo"
$IMAGE_BASE = "$REGION-docker.pkg.dev/$PROJECT_ID/$REPO"
```

## Deploy Services

### 1. Deploy Orchestrator (Public)

```powershell
gcloud run deploy explainiq-orchestrator `
  --image "$IMAGE_BASE/explainiq-orchestrator:latest" `
  --platform managed `
  --region $REGION `
  --allow-unauthenticated `
  --port 8080 `
  --memory 2Gi `
  --cpu 2 `
  --min-instances 1 `
  --max-instances 10 `
  --timeout 300 `
  --concurrency 100 `
  --set-env-vars "GIN_MODE=release,LOG_LEVEL=info"
```

### 2. Deploy Summarizer (Internal)

```powershell
gcloud run deploy explainiq-summarizer `
  --image "$IMAGE_BASE/explainiq-summarizer:latest" `
  --platform managed `
  --region $REGION `
  --no-allow-unauthenticated `
  --port 8081 `
  --memory 4Gi `
  --cpu 4 `
  --min-instances 0 `
  --max-instances 10 `
  --timeout 600 `
  --concurrency 50 `
  --set-env-vars "GIN_MODE=release,LOG_LEVEL=info" `
  --set-secrets "GEMINI_API_KEY=google-ai-api-key:latest"
```

### 3. Deploy Explainer (Internal)

```powershell
gcloud run deploy explainiq-explainer `
  --image "$IMAGE_BASE/explainiq-explainer:latest" `
  --platform managed `
  --region $REGION `
  --no-allow-unauthenticated `
  --port 8082 `
  --memory 4Gi `
  --cpu 4 `
  --min-instances 0 `
  --max-instances 10 `
  --timeout 600 `
  --concurrency 50 `
  --set-env-vars "GIN_MODE=release,LOG_LEVEL=info" `
  --set-secrets "GEMINI_API_KEY=google-ai-api-key:latest"
```

### 4. Deploy Critic (Internal)

```powershell
gcloud run deploy explainiq-critic `
  --image "$IMAGE_BASE/explainiq-critic:latest" `
  --platform managed `
  --region $REGION `
  --no-allow-unauthenticated `
  --port 8083 `
  --memory 4Gi `
  --cpu 4 `
  --min-instances 0 `
  --max-instances 10 `
  --timeout 600 `
  --concurrency 50 `
  --set-env-vars "GIN_MODE=release,LOG_LEVEL=info" `
  --set-secrets "GEMINI_API_KEY=google-ai-api-key:latest"
```

### 5. Deploy Visualizer (Internal)

```powershell
gcloud run deploy explainiq-visualizer `
  --image "$IMAGE_BASE/explainiq-visualizer:latest" `
  --platform managed `
  --region $REGION `
  --no-allow-unauthenticated `
  --port 8084 `
  --memory 4Gi `
  --cpu 4 `
  --min-instances 0 `
  --max-instances 10 `
  --timeout 600 `
  --concurrency 50 `
  --set-env-vars "GIN_MODE=release,LOG_LEVEL=info" `
  --set-secrets "GEMINI_API_KEY=google-ai-api-key:latest"
```

### 6. Deploy Frontend (Public)

```powershell
gcloud run deploy explainiq-frontend `
  --image "$IMAGE_BASE/explainiq-frontend:latest" `
  --platform managed `
  --region $REGION `
  --allow-unauthenticated `
  --port 8085 `
  --memory 2Gi `
  --cpu 2 `
  --min-instances 1 `
  --max-instances 10 `
  --timeout 300 `
  --concurrency 100 `
  --set-env-vars "GIN_MODE=release,LOG_LEVEL=info"
```

## Configure Service-to-Service Authentication

### Get Service URLs

```powershell
$ORCHESTRATOR_URL = gcloud run services describe explainiq-orchestrator --region=$REGION --format="value(status.url)"
$SUMMARIZER_URL = gcloud run services describe explainiq-summarizer --region=$REGION --format="value(status.url)"
$EXPLAINER_URL = gcloud run services describe explainiq-explainer --region=$REGION --format="value(status.url)"
$CRITIC_URL = gcloud run services describe explainiq-critic --region=$REGION --format="value(status.url)"
$VISUALIZER_URL = gcloud run services describe explainiq-visualizer --region=$REGION --format="value(status.url)"
```

### Grant Orchestrator Permission to Invoke Agents

```powershell
# Get orchestrator service account
$ORCHESTRATOR_SA = gcloud run services describe explainiq-orchestrator --region=$REGION --format="value(spec.template.spec.serviceAccountName)"
if ([string]::IsNullOrEmpty($ORCHESTRATOR_SA)) {
    $ORCHESTRATOR_SA = "$PROJECT_ID@$PROJECT_ID.iam.gserviceaccount.com"
}

# Grant permissions
gcloud run services add-iam-policy-binding explainiq-summarizer `
  --region=$REGION `
  --member="serviceAccount:$ORCHESTRATOR_SA" `
  --role="roles/run.invoker"

gcloud run services add-iam-policy-binding explainiq-explainer `
  --region=$REGION `
  --member="serviceAccount:$ORCHESTRATOR_SA" `
  --role="roles/run.invoker"

gcloud run services add-iam-policy-binding explainiq-critic `
  --region=$REGION `
  --member="serviceAccount:$ORCHESTRATOR_SA" `
  --role="roles/run.invoker"

gcloud run services add-iam-policy-binding explainiq-visualizer `
  --region=$REGION `
  --member="serviceAccount:$ORCHESTRATOR_SA" `
  --role="roles/run.invoker"
```

### Update Orchestrator Environment Variables

```powershell
gcloud run services update explainiq-orchestrator `
  --region=$REGION `
  --update-env-vars "AGENT_SUMMARIZER_URL=$SUMMARIZER_URL,AGENT_EXPLAINER_URL=$EXPLAINER_URL,AGENT_CRITIC_URL=$CRITIC_URL,AGENT_VISUALIZER_URL=$VISUALIZER_URL,GCP_PROJECT_ID=$PROJECT_ID,SERVICE_URL=$ORCHESTRATOR_URL"
```

### Update Frontend Environment Variables

```powershell
gcloud run services update explainiq-frontend `
  --region=$REGION `
  --update-env-vars "NEXT_PUBLIC_ORCHESTRATOR_URL=$ORCHESTRATOR_URL,GCS_PROJECT_ID=$PROJECT_ID"
```

## Verify Deployment

```powershell
# List all services
gcloud run services list --region=$REGION

# Get service URLs
gcloud run services describe explainiq-orchestrator --region=$REGION --format="value(status.url)"
gcloud run services describe explainiq-frontend --region=$REGION --format="value(status.url)"
```

## Notes

- Replace `YOUR_GEMINI_API_KEY` with your actual Gemini API key
- All agent services (summarizer, explainer, critic, visualizer) require the `google-ai-api-key` secret
- Agent services are internal (no public access) - only orchestrator can invoke them
- Orchestrator and frontend are public (allow-unauthenticated)
- Make sure images are already built and pushed to Artifact Registry before deploying

