# Quick redeploy script for orchestrator service only
# This redeploys the orchestrator with the fixed JSON-RPC method name

param(
    [Parameter(Mandatory=$true)]
    [string]$ProjectId,
    
    [string]$Region = "europe-west1",
    [string]$Repo = "explainiq-repo"
)

$ErrorActionPreference = "Stop"

Write-Host "Redeploying orchestrator service with JSON-RPC fix..." -ForegroundColor Green
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

# Build and push orchestrator image
$ImageBase = "$Region-docker.pkg.dev/$ProjectId/$Repo"
$ImageName = "$ImageBase/explainiq-orchestrator:latest"

Write-Host "Building orchestrator Docker image..." -ForegroundColor Yellow
docker build -f docker/Dockerfile.orchestrator -t $ImageName .

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Docker build failed" -ForegroundColor Red
    exit 1
}

Write-Host "Pushing orchestrator Docker image..." -ForegroundColor Yellow
docker push $ImageName

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Docker push failed" -ForegroundColor Red
    exit 1
}

Write-Host "Image pushed successfully" -ForegroundColor Green

# Update orchestrator service with new image
Write-Host "Updating orchestrator Cloud Run service..." -ForegroundColor Yellow
gcloud run services update explainiq-orchestrator `
    --image=$ImageName `
    --region=$Region `
    --platform=managed `
    --quiet

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to update orchestrator service" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Orchestrator service redeployed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "The fix changes the JSON-RPC method name from execute to invoke to match the A2A protocol." -ForegroundColor Cyan
Write-Host "This should resolve the method not found error with code -32601." -ForegroundColor Cyan

