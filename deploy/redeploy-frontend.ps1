# Quick redeploy script for frontend service only
# This redeploys the frontend with the latest code including API routes

param(
    [Parameter(Mandatory=$true)]
    [string]$ProjectId,
    
    [string]$Region = "europe-west1",
    [string]$Repo = "explainiq-repo"
)

$ErrorActionPreference = "Stop"

Write-Host "Redeploying frontend service..." -ForegroundColor Green
Write-Host "Project ID: $ProjectId" -ForegroundColor Cyan
Write-Host "Region: $Region" -ForegroundColor Cyan
Write-Host "Repository: $Repo" -ForegroundColor Cyan
Write-Host ""

# Set project
Write-Host "Setting GCP project..." -ForegroundColor Yellow
gcloud config set project $ProjectId

# Configure Docker authentication
Write-Host "Configuring Docker authentication..." -ForegroundColor Yellow
gcloud auth configure-docker "$Region-docker.pkg.dev" --quiet

# Get orchestrator URL for environment variable
Write-Host "Getting orchestrator URL..." -ForegroundColor Yellow
$OrchestratorUrl = gcloud run services describe explainiq-orchestrator --region $Region --format "value(status.url)" --project $ProjectId

if ([string]::IsNullOrEmpty($OrchestratorUrl)) {
    Write-Host "WARNING: Could not get orchestrator URL. Using default." -ForegroundColor Yellow
    $OrchestratorUrl = "https://explainiq-orchestrator-othekugkka-ew.a.run.app"
}

Write-Host "Orchestrator URL: $OrchestratorUrl" -ForegroundColor Cyan

# Build and push frontend image
$ImageBase = "$Region-docker.pkg.dev/$ProjectId/$Repo"
$ImageName = "$ImageBase/explainiq-frontend:latest"

Write-Host "Building frontend Docker image..." -ForegroundColor Yellow
docker build `
    -f cmd/frontend/nextjs/Dockerfile `
    --build-arg NEXT_PUBLIC_ORCHESTRATOR_URL=$OrchestratorUrl `
    --build-arg ORCHESTRATOR_URL=$OrchestratorUrl `
    --build-arg GCS_PROJECT_ID=$ProjectId `
    --build-arg GCS_BUCKET="explainiq-pdfs" `
    -t $ImageName `
    cmd/frontend/nextjs

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Docker build failed" -ForegroundColor Red
    exit 1
}

Write-Host "Pushing frontend Docker image..." -ForegroundColor Yellow
docker push $ImageName

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Docker push failed" -ForegroundColor Red
    exit 1
}

Write-Host "Image pushed successfully" -ForegroundColor Green

# Update frontend service with new image and environment variables
Write-Host "Updating frontend Cloud Run service..." -ForegroundColor Yellow
gcloud run services update explainiq-frontend `
    --image=$ImageName `
    --region=$Region `
    --platform=managed `
    --update-env-vars "ORCHESTRATOR_URL=$OrchestratorUrl,NEXT_PUBLIC_ORCHESTRATOR_URL=$OrchestratorUrl,GCS_PROJECT_ID=$ProjectId,GCS_BUCKET=explainiq-pdfs" `
    --quiet

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to update frontend service" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Frontend service redeployed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "The frontend API routes should now be available, including /api/sessions/[id]/run for SSE." -ForegroundColor Cyan

