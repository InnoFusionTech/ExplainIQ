# Script to set up Google AI API Key secret for Cloud Run services
# Usage: .\deploy\setup-secret.ps1 -ProjectId "explainiq-477714" -ApiKey "YOUR_API_KEY"

param(
    [Parameter(Mandatory=$true)]
    [string]$ProjectId,
    
    [Parameter(Mandatory=$true)]
    [string]$ApiKey,
    
    [string]$Region = "europe-west1"
)

$ErrorActionPreference = "Stop"

Write-Host "Setting up Google AI API Key secret..." -ForegroundColor Green
Write-Host "Project ID: $ProjectId" -ForegroundColor Yellow
Write-Host ""

# Check if secret exists
$secretExists = $false
try {
    $null = gcloud secrets describe google-ai-api-key --project=$ProjectId 2>&1 | Out-Null
    $secretExists = $true
    Write-Host "Secret 'google-ai-api-key' already exists" -ForegroundColor Green
} catch {
    Write-Host "Creating secret 'google-ai-api-key'..." -ForegroundColor Yellow
    $null = gcloud secrets create google-ai-api-key --replication-policy="automatic" --project=$ProjectId 2>&1 | Out-Null
    Write-Host "Secret created successfully" -ForegroundColor Green
}

# Set the secret value
Write-Host "Setting secret value..." -ForegroundColor Yellow
$ApiKey | gcloud secrets versions add google-ai-api-key --data-file=- --project=$ProjectId 2>&1 | Out-Null
Write-Host "Secret value set successfully" -ForegroundColor Green

# Grant Cloud Run service accounts access to the secret
Write-Host ""
Write-Host "Granting Cloud Run services access to the secret..." -ForegroundColor Yellow

# Get the default compute service account (use project number, not project ID)
# First, get the project number
Write-Host "Getting project number..." -ForegroundColor Yellow
$projectNumber = gcloud projects describe $ProjectId --format="value(projectNumber)" 2>&1
$projectNumber = $projectNumber.Trim()
$computeSa = "$projectNumber-compute@developer.gserviceaccount.com"

# Grant secret accessor role to the compute service account
Write-Host "Granting access to compute service account: $computeSa" -ForegroundColor Cyan
$null = gcloud secrets add-iam-policy-binding google-ai-api-key `
    --member="serviceAccount:$computeSa" `
    --role="roles/secretmanager.secretAccessor" `
    --project=$ProjectId `
    --quiet 2>&1 | Out-Null

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "Secret setup complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "The secret 'google-ai-api-key' is now configured and accessible by Cloud Run services." -ForegroundColor Green
Write-Host ""
Write-Host "To update the secret value in the future, run:" -ForegroundColor Yellow
Write-Host "  echo 'YOUR_NEW_API_KEY' | gcloud secrets versions add google-ai-api-key --data-file=- --project=$ProjectId" -ForegroundColor Cyan
Write-Host ""

