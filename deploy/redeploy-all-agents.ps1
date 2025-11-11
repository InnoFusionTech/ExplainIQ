# Quick redeploy script for all agent services
# This redeploys all agent services (summarizer, explainer, critic, visualizer)

param(
    [Parameter(Mandatory=$true)]
    [string]$ProjectId,
    
    [string]$Region = "europe-west1",
    [string]$Repo = "explainiq-repo"
)

$ErrorActionPreference = "Stop"

Write-Host "Redeploying all agent services..." -ForegroundColor Green
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

# Image base
$ImageBase = "$Region-docker.pkg.dev/$ProjectId/$Repo"

# Agent services to deploy
$agents = @(
    @{Name="summarizer"; Dockerfile="Dockerfile.agent-summarizer"; ImageName="explainiq-summarizer"},
    @{Name="explainer"; Dockerfile="Dockerfile.agent-explainer"; ImageName="explainiq-explainer"},
    @{Name="critic"; Dockerfile="Dockerfile.agent-critic"; ImageName="explainiq-critic"},
    @{Name="visualizer"; Dockerfile="Dockerfile.agent-visualizer"; ImageName="explainiq-visualizer"}
)

foreach ($agent in $agents) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "Deploying $($agent.Name)..." -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    
    $ImageName = "$ImageBase/$($agent.ImageName):latest"
    
    # Build Docker image
    Write-Host "Building $($agent.Name) Docker image..." -ForegroundColor Yellow
    docker build -f "docker/$($agent.Dockerfile)" -t $ImageName .
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Docker build failed for $($agent.Name)" -ForegroundColor Red
        continue
    }
    
    # Push Docker image
    Write-Host "Pushing $($agent.Name) Docker image..." -ForegroundColor Yellow
    docker push $ImageName
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Docker push failed for $($agent.Name)" -ForegroundColor Red
        continue
    }
    
    Write-Host "Image pushed successfully" -ForegroundColor Green
    
    # Update Cloud Run service
    Write-Host "Updating $($agent.Name) Cloud Run service..." -ForegroundColor Yellow
    gcloud run services update $($agent.ImageName) `
        --image=$ImageName `
        --region=$Region `
        --platform=managed `
        --quiet
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Failed to update $($agent.Name) service" -ForegroundColor Red
        continue
    }
    
    Write-Host "[OK] $($agent.Name) service redeployed successfully!" -ForegroundColor Green
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "All agent services redeployed!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Services redeployed:" -ForegroundColor Cyan
foreach ($agent in $agents) {
    Write-Host "  - $($agent.Name)" -ForegroundColor Cyan
}
Write-Host ""

