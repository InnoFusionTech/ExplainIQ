# Google Cloud Run Deployment Script (PowerShell)
# Deploys all ExplainIQ services to Cloud Run

param(
    [Parameter(Mandatory=$true)]
    [string]$ProjectId,
    
    [string]$Region = "europe-west1",
    [string]$Repo = "explainiq-repo",
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

# Suppress informational stderr messages from gcloud
$env:GCLOUD_PYTHON_OUTPUT_TO_STDERR = "0"

# Helper function to run gcloud commands without PowerShell error formatting
function Invoke-GcloudCommand {
    param(
        [string[]]$Arguments,
        [switch]$SuppressOutput
    )
    
    # Temporarily disable error action to prevent PowerShell from formatting gcloud output as errors
    $oldErrorAction = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    
    # Run the command and capture all output, suppressing PowerShell error formatting
    $output = ""
    $exitCode = 0
    
    # Use cmd.exe to run gcloud directly, bypassing PowerShell wrapper to avoid error formatting
    $argString = ($Arguments | ForEach-Object { if ($_ -match '\s') { "`"$_`"" } else { $_ } }) -join " "
    $process = Start-Process -FilePath "cmd.exe" -ArgumentList "/c", "gcloud $argString" -NoNewWindow -PassThru -Wait -RedirectStandardOutput "$env:TEMP\gcloud_stdout.txt" -RedirectStandardError "$env:TEMP\gcloud_stderr.txt"
    $exitCode = $process.ExitCode
    
    # Read captured output
    $stdout = ""
    $stderr = ""
    if (Test-Path "$env:TEMP\gcloud_stdout.txt") {
        $stdout = Get-Content "$env:TEMP\gcloud_stdout.txt" -Raw -ErrorAction SilentlyContinue
        Remove-Item "$env:TEMP\gcloud_stdout.txt" -ErrorAction SilentlyContinue
    }
    if (Test-Path "$env:TEMP\gcloud_stderr.txt") {
        $stderr = Get-Content "$env:TEMP\gcloud_stderr.txt" -Raw -ErrorAction SilentlyContinue
        Remove-Item "$env:TEMP\gcloud_stderr.txt" -ErrorAction SilentlyContinue
    }
    
    # Combine output (gcloud writes info to stderr)
    $output = if ($stdout) { $stdout.Trim() } else { "" }
    if ($stderr) {
        $output += if ($output) { "`n" + $stderr.Trim() } else { $stderr.Trim() }
    }
    
    # Restore error action
    $ErrorActionPreference = $oldErrorAction
    
    # Only show output if there's an actual error (non-zero exit code) and not suppressing
    if ($exitCode -ne 0 -and -not $SuppressOutput) {
        Write-Host $output -ForegroundColor Red
        throw "gcloud command failed with exit code $exitCode"
    }
    
    return $output
}

Write-Host "Deploying ExplainIQ to Google Cloud Run" -ForegroundColor Green
Write-Host "Project ID: $ProjectId"
Write-Host "Region: $Region"
Write-Host "Repository: $Repo"
Write-Host ""

# Set project
Write-Host "Setting GCP project..." -ForegroundColor Yellow
$null = Invoke-GcloudCommand -Arguments @("config", "set", "project", $ProjectId) -SuppressOutput

# Enable required APIs
Write-Host "Enabling required APIs..." -ForegroundColor Yellow
$null = Invoke-GcloudCommand -Arguments @("services", "enable", "run.googleapis.com", "cloudbuild.googleapis.com", "artifactregistry.googleapis.com", "secretmanager.googleapis.com", "storage.googleapis.com", "--quiet") -SuppressOutput

# Check if secret exists and grant access if needed
Write-Host "Checking secret access..." -ForegroundColor Yellow
try {
    $secretExists = Invoke-GcloudCommand -Arguments @("secrets", "describe", "google-ai-api-key", "--project=$ProjectId") -SuppressOutput
    if ($secretExists) {
        # Get project number for service account (Cloud Run uses project number, not project ID)
        $projectNumber = Invoke-GcloudCommand -Arguments @("projects", "describe", $ProjectId, "--format=value(projectNumber)") -SuppressOutput
        $projectNumber = $projectNumber.Trim()
        $computeSa = "$projectNumber-compute@developer.gserviceaccount.com"
        
        # Grant Cloud Run service accounts access to the secret
        Write-Host "Granting secret access to Cloud Run services..." -ForegroundColor Yellow
        $null = Invoke-GcloudCommand -Arguments @("secrets", "add-iam-policy-binding", "google-ai-api-key", "--member=serviceAccount:$computeSa", "--role=roles/secretmanager.secretAccessor", "--project=$ProjectId", "--quiet") -SuppressOutput
    }
} catch {
    Write-Host "WARNING: Secret 'google-ai-api-key' does not exist. Create it using:" -ForegroundColor Yellow
    Write-Host "  .\deploy\setup-secret.ps1 -ProjectId $ProjectId -ApiKey YOUR_API_KEY" -ForegroundColor Cyan
}

# Create Artifact Registry repository if it doesn't exist
Write-Host "Setting up Artifact Registry..." -ForegroundColor Yellow
try {
    $null = Invoke-GcloudCommand -Arguments @("artifacts", "repositories", "describe", $Repo, "--location=$Region") -SuppressOutput
} catch {
    # Repository doesn't exist, create it
    try {
        $null = Invoke-GcloudCommand -Arguments @("artifacts", "repositories", "create", $Repo, "--repository-format=docker", "--location=$Region", "--description=ExplainIQ Docker images", "--quiet") -SuppressOutput
    } catch {
        # Repository might already exist, ignore error
    }
}

# Create Cloud Storage buckets if they don't exist
Write-Host "Setting up Cloud Storage buckets..." -ForegroundColor Yellow

# Bucket for diagrams/images
$diagramsBucket = "explainiq-diagrams"
try {
    $null = Invoke-GcloudCommand -Arguments @("storage", "buckets", "describe", "gs://$diagramsBucket", "--project=$ProjectId") -SuppressOutput
    Write-Host "  Bucket '$diagramsBucket' already exists" -ForegroundColor Green
} catch {
    # Bucket doesn't exist, create it
    try {
        Write-Host "  Creating bucket '$diagramsBucket'..." -ForegroundColor Cyan
        $null = Invoke-GcloudCommand -Arguments @("storage", "buckets", "create", "gs://$diagramsBucket", "--project=$ProjectId", "--location=$Region", "--uniform-bucket-level-access") -SuppressOutput
        Write-Host "  [OK] Bucket '$diagramsBucket' created successfully" -ForegroundColor Green
    } catch {
        Write-Host "  WARNING: Failed to create bucket '$diagramsBucket'. You may need to create it manually:" -ForegroundColor Yellow
        Write-Host "    gsutil mb -p $ProjectId -l $Region gs://$diagramsBucket" -ForegroundColor Cyan
    }
}

# Bucket for PDFs
$pdfsBucket = "explainiq-pdfs"
try {
    $null = Invoke-GcloudCommand -Arguments @("storage", "buckets", "describe", "gs://$pdfsBucket", "--project=$ProjectId") -SuppressOutput
    Write-Host "  Bucket '$pdfsBucket' already exists" -ForegroundColor Green
} catch {
    # Bucket doesn't exist, create it
    try {
        Write-Host "  Creating bucket '$pdfsBucket'..." -ForegroundColor Cyan
        $null = Invoke-GcloudCommand -Arguments @("storage", "buckets", "create", "gs://$pdfsBucket", "--project=$ProjectId", "--location=$Region", "--uniform-bucket-level-access") -SuppressOutput
        Write-Host "  [OK] Bucket '$pdfsBucket' created successfully" -ForegroundColor Green
    } catch {
        Write-Host "  WARNING: Failed to create bucket '$pdfsBucket'. You may need to create it manually:" -ForegroundColor Yellow
        Write-Host "    gsutil mb -p $ProjectId -l $Region gs://$pdfsBucket" -ForegroundColor Cyan
    }
}

# Configure Docker authentication
Write-Host "Configuring Docker authentication..." -ForegroundColor Yellow
try {
    $null = Invoke-GcloudCommand -Arguments @("auth", "configure-docker", "$Region-docker.pkg.dev", "--quiet") -SuppressOutput
} catch {
    # Ignore errors from informational messages
}

# Build and push images
if (-not $SkipBuild) {
    Write-Host "Building and pushing Docker images..." -ForegroundColor Yellow
    
    $ImageBase = "$Region-docker.pkg.dev/$ProjectId/$Repo"
    
    # Build orchestrator
    Write-Host "Building orchestrator..."
    docker build -f docker/Dockerfile.orchestrator -t "$ImageBase/explainiq-orchestrator:latest" .
    docker push "$ImageBase/explainiq-orchestrator:latest"
    
    # Build agents
    Write-Host "Building agent-summarizer..."
    docker build -f docker/Dockerfile.agent-summarizer -t "$ImageBase/explainiq-summarizer:latest" .
    docker push "$ImageBase/explainiq-summarizer:latest"
    
    Write-Host "Building agent-explainer..."
    docker build -f docker/Dockerfile.agent-explainer -t "$ImageBase/explainiq-explainer:latest" .
    docker push "$ImageBase/explainiq-explainer:latest"
    
    Write-Host "Building agent-critic..."
    docker build -f docker/Dockerfile.agent-critic -t "$ImageBase/explainiq-critic:latest" .
    docker push "$ImageBase/explainiq-critic:latest"
    
    Write-Host "Building agent-visualizer..."
    docker build -f docker/Dockerfile.agent-visualizer -t "$ImageBase/explainiq-visualizer:latest" .
    docker push "$ImageBase/explainiq-visualizer:latest"
    
    Write-Host "Building frontend..."
    docker build -f cmd/frontend/nextjs/Dockerfile -t "$ImageBase/explainiq-frontend:latest" cmd/frontend/nextjs
    docker push "$ImageBase/explainiq-frontend:latest"
    
    Write-Host "All images built and pushed successfully" -ForegroundColor Green
}

# Update YAML files with project ID and image paths
Write-Host "Updating Cloud Run YAML files..." -ForegroundColor Yellow
$ImageBase = "$Region-docker.pkg.dev/$ProjectId/$Repo"

# Create temporary directory for updated YAMLs
$TmpDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_.FullName }
Copy-Item deploy/cloudrun/*.yaml $TmpDir

# Replace placeholders in all YAML files
# IMPORTANT: Replace full path first, then individual placeholders
# This prevents double replacement of REPO
Get-ChildItem $TmpDir/*.yaml | ForEach-Object {
    $content = Get-Content $_.FullName -Raw
    # Replace full image path first (this replaces REGION, PROJECT_ID, and REPO in the path)
    $content = $content -replace "REGION-docker.pkg.dev/PROJECT_ID/REPO", $ImageBase
    # Then replace remaining standalone placeholders (not in image paths)
    # Replace PROJECT_ID only if it's not part of an image path
    $content = $content -replace "PROJECT_ID", $ProjectId
    # Replace REGION only if it's not part of an image path
    $content = $content -replace "REGION", $Region
    # Don't replace REPO - it's already been replaced in the image path
    # Only replace if it appears standalone (not in a path)
    # Match REPO only when it's not preceded by / or followed by /
    $content = $content -replace "(?<![/\w])REPO(?![/\w])", $Repo
    # Replace ORCHESTRATOR_URL_PLACEHOLDER with a temporary value (will be updated after deployment)
    # We'll use a placeholder that won't conflict with actual URLs
    $content = $content -replace "ORCHESTRATOR_URL_PLACEHOLDER", "https://explainiq-orchestrator-PLACEHOLDER.a.run.app"
    # Replace agent URL placeholders (will be updated after deployment)
    $content = $content -replace "AGENT_SUMMARIZER_URL_PLACEHOLDER", "https://explainiq-summarizer-PLACEHOLDER.a.run.app"
    $content = $content -replace "AGENT_EXPLAINER_URL_PLACEHOLDER", "https://explainiq-explainer-PLACEHOLDER.a.run.app"
    $content = $content -replace "AGENT_CRITIC_URL_PLACEHOLDER", "https://explainiq-critic-PLACEHOLDER.a.run.app"
    $content = $content -replace "AGENT_VISUALIZER_URL_PLACEHOLDER", "https://explainiq-visualizer-PLACEHOLDER.a.run.app"
    $content = $content -replace "SERVICE_URL_PLACEHOLDER", "https://explainiq-orchestrator-PLACEHOLDER.a.run.app"
    Set-Content $_.FullName -Value $content -NoNewline
}

# Deploy services
Write-Host "Deploying Cloud Run services..." -ForegroundColor Yellow

# Function to deploy a service with error handling
function Deploy-Service {
    param(
        [string]$ServiceName,
        [string]$YamlFile,
        [string]$Region,
        [bool]$AllowUnauthenticated = $true
    )
    
    Write-Host "Deploying $ServiceName..." -ForegroundColor Cyan
    
    # Note: gcloud run services replace does NOT support --allow-unauthenticated flag
    # We need to set IAM policy after deployment
    $deployArgs = @(
        "run", "services", "replace", $YamlFile,
        "--region=$Region",
        "--platform=managed",
        "--quiet"
    )
    
    # Use helper function to suppress PowerShell error formatting
    try {
        $output = Invoke-GcloudCommand -Arguments $deployArgs -SuppressOutput:$false
        Write-Host "[OK] $ServiceName deployed successfully" -ForegroundColor Green
        
        # Set IAM policy for public/private access after deployment
        if ($AllowUnauthenticated) {
            Write-Host "Granting public access to $ServiceName..." -ForegroundColor Yellow
            $null = Invoke-GcloudCommand -Arguments @("run", "services", "add-iam-policy-binding", $ServiceName, "--region=$Region", "--member=allUsers", "--role=roles/run.invoker", "--quiet") -SuppressOutput
        } else {
            Write-Host "Ensuring $ServiceName is private (no public access)..." -ForegroundColor Yellow
            # Remove allUsers access if it exists (ignore errors if it doesn't exist)
            try {
                $null = Invoke-GcloudCommand -Arguments @("run", "services", "remove-iam-policy-binding", $ServiceName, "--region=$Region", "--member=allUsers", "--role=roles/run.invoker", "--quiet") -SuppressOutput
            } catch {
                # Ignore errors - the binding might not exist
            }
        }
    } catch {
        Write-Host "ERROR: Failed to deploy $ServiceName" -ForegroundColor Red
        Write-Host $_.Exception.Message -ForegroundColor Red
        throw
    }
}

# Deploy orchestrator (public)
try {
    Deploy-Service -ServiceName "explainiq-orchestrator" -YamlFile "$TmpDir/orchestrator.yaml" -Region $Region -AllowUnauthenticated $true
} catch {
    Write-Host "ERROR: Failed to deploy orchestrator. Aborting deployment." -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
    exit 1
}

# Deploy agents (internal - no public ingress)
$agentServices = @(
    @{Name="explainiq-summarizer"; File="summarizer.yaml"},
    @{Name="explainiq-explainer"; File="explainer.yaml"},
    @{Name="explainiq-critic"; File="critic.yaml"},
    @{Name="explainiq-visualizer"; File="visualizer.yaml"}
)

foreach ($agent in $agentServices) {
    try {
        Write-Host "Deploying $($agent.Name)..." -ForegroundColor Cyan
        Deploy-Service -ServiceName $agent.Name -YamlFile "$TmpDir/$($agent.File)" -Region $Region -AllowUnauthenticated $false
        Write-Host "[OK] $($agent.Name) deployed successfully" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Failed to deploy $($agent.Name)" -ForegroundColor Red
        Write-Host "Error details: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "Full error: $($_.Exception)" -ForegroundColor Red
        Write-Host "Attempting to get more details..." -ForegroundColor Yellow
        
        # Try to get more information about why it failed
        $checkOutput = Invoke-GcloudCommand -Arguments @("run", "services", "describe", $agent.Name, "--region=$Region", "--format=json") -SuppressOutput 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Service does not exist. Checking if image exists..." -ForegroundColor Yellow
            $imageName = "$Region-docker.pkg.dev/$ProjectId/$Repo/$($agent.Name):latest"
            $imageCheck = docker images "$imageName" --format "{{.Repository}}:{{.Tag}}" 2>&1
            Write-Host "Image check: $imageCheck" -ForegroundColor Yellow
        }
        
        Write-Host "Continuing with other services..." -ForegroundColor Yellow
    }
}

# Deploy frontend (public)
try {
    Deploy-Service -ServiceName "explainiq-frontend" -YamlFile "$TmpDir/frontend.yaml" -Region $Region -AllowUnauthenticated $true
} catch {
    Write-Host "WARNING: Failed to deploy frontend. Continuing..." -ForegroundColor Yellow
    Write-Host $_.Exception.Message -ForegroundColor Yellow
}

# Cleanup
Remove-Item -Recurse -Force $TmpDir

# Get service URLs with error handling
Write-Host "Getting service URLs..." -ForegroundColor Yellow

function Get-ServiceUrl {
    param(
        [string]$ServiceName,
        [string]$Region
    )
    
    $url = Invoke-GcloudCommand -Arguments @("run", "services", "describe", $ServiceName, "--region=$Region", "--format=value(status.url)") -SuppressOutput
    $url = $url.Trim()
    
    if ([string]::IsNullOrEmpty($url)) {
        Write-Host "  Warning: Could not get URL for $ServiceName" -ForegroundColor Yellow
        return $null
    }
    
    return $url
}

$OrchestratorUrl = Get-ServiceUrl -ServiceName "explainiq-orchestrator" -Region $Region
$SummarizerUrl = Get-ServiceUrl -ServiceName "explainiq-summarizer" -Region $Region
$ExplainerUrl = Get-ServiceUrl -ServiceName "explainiq-explainer" -Region $Region
$CriticUrl = Get-ServiceUrl -ServiceName "explainiq-critic" -Region $Region
$VisualizerUrl = Get-ServiceUrl -ServiceName "explainiq-visualizer" -Region $Region
$FrontendUrl = Get-ServiceUrl -ServiceName "explainiq-frontend" -Region $Region

# Verify critical services are deployed
if ([string]::IsNullOrEmpty($OrchestratorUrl)) {
    Write-Host "ERROR: Orchestrator service is not deployed or not accessible" -ForegroundColor Red
    exit 1
}

if ([string]::IsNullOrEmpty($SummarizerUrl) -or [string]::IsNullOrEmpty($ExplainerUrl) -or 
    [string]::IsNullOrEmpty($CriticUrl) -or [string]::IsNullOrEmpty($VisualizerUrl)) {
    Write-Host "WARNING: Some agent services are not deployed or not accessible" -ForegroundColor Yellow
}

# Configure service-to-service authentication
Write-Host "Configuring service-to-service authentication..." -ForegroundColor Yellow

# Get orchestrator service account
$OrchestratorSa = Invoke-GcloudCommand -Arguments @("run", "services", "describe", "explainiq-orchestrator", "--region=$Region", "--format=value(spec.template.spec.serviceAccountName)") -SuppressOutput
$OrchestratorSa = $OrchestratorSa.Trim()

if ([string]::IsNullOrEmpty($OrchestratorSa)) {
    # Use default compute service account
    $OrchestratorSa = "$ProjectId-compute@developer.gserviceaccount.com"
    Write-Host "Using default compute service account: $OrchestratorSa" -ForegroundColor Yellow
}

# Grant orchestrator permission to invoke agents
$services = @("explainiq-summarizer", "explainiq-explainer", "explainiq-critic", "explainiq-visualizer")
foreach ($service in $services) {
    Write-Host "Granting access to $service..."
    $null = Invoke-GcloudCommand -Arguments @("run", "services", "add-iam-policy-binding", $service, "--region=$Region", "--member=serviceAccount:$OrchestratorSa", "--role=roles/run.invoker", "--quiet") -SuppressOutput
}

# Function to update service environment variables
function Update-ServiceEnvVars {
    param(
        [string]$ServiceName,
        [string]$EnvVars,
        [string]$Region
    )
    
    try {
        $null = Invoke-GcloudCommand -Arguments @("run", "services", "update", $ServiceName, "--region=$Region", "--update-env-vars=$EnvVars", "--quiet") -SuppressOutput
        Write-Host "[OK] $ServiceName environment variables updated" -ForegroundColor Green
    } catch {
        Write-Host "WARNING: Failed to update $ServiceName environment variables" -ForegroundColor Yellow
        Write-Host $_.Exception.Message -ForegroundColor Yellow
    }
}

# Update orchestrator with agent URLs (only if all URLs are available)
$allAgentUrlsAvailable = (-not [string]::IsNullOrEmpty($SummarizerUrl)) -and (-not [string]::IsNullOrEmpty($ExplainerUrl)) -and (-not [string]::IsNullOrEmpty($CriticUrl)) -and (-not [string]::IsNullOrEmpty($VisualizerUrl))

if ($allAgentUrlsAvailable) {
    Write-Host "Updating orchestrator environment variables..." -ForegroundColor Yellow
    $envVars = "AGENT_SUMMARIZER_URL=$SummarizerUrl,AGENT_EXPLAINER_URL=$ExplainerUrl,AGENT_CRITIC_URL=$CriticUrl,AGENT_VISUALIZER_URL=$VisualizerUrl,GCP_PROJECT_ID=$ProjectId,SERVICE_URL=$OrchestratorUrl"
    Update-ServiceEnvVars -ServiceName "explainiq-orchestrator" -EnvVars $envVars -Region $Region | Out-Null
} else {
    Write-Host "WARNING: Skipping orchestrator environment variable update - some agent URLs are missing" -ForegroundColor Yellow
}

# Update frontend with orchestrator URL (only if orchestrator URL is available)
$frontendUrlsAvailable = (-not [string]::IsNullOrEmpty($OrchestratorUrl)) -and (-not [string]::IsNullOrEmpty($FrontendUrl))

if ($frontendUrlsAvailable) {
    Write-Host "Updating frontend environment variables..." -ForegroundColor Yellow
    $envVars = "ORCHESTRATOR_URL=$OrchestratorUrl,NEXT_PUBLIC_ORCHESTRATOR_URL=$OrchestratorUrl,GCS_PROJECT_ID=$ProjectId"
    Update-ServiceEnvVars -ServiceName "explainiq-frontend" -EnvVars $envVars -Region $Region | Out-Null
} else {
    Write-Host "WARNING: Skipping frontend environment variable update - orchestrator or frontend URL is missing" -ForegroundColor Yellow
}

# Display deployment summary
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "Deployment Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Service URLs:"
Write-Host "  Orchestrator: $OrchestratorUrl"
Write-Host "  Frontend:     $FrontendUrl"
Write-Host ""
Write-Host "Agent URLs (internal):"
Write-Host "  Summarizer:    $SummarizerUrl"
Write-Host "  Explainer:   $ExplainerUrl"
Write-Host "  Critic:      $CriticUrl"
Write-Host "  Visualizer:  $VisualizerUrl"
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Yellow
Write-Host "1. Verify services are running:"
Write-Host "   gcloud run services list --region=$($Region)"
Write-Host ""
Write-Host "2. Test the frontend:"
if (-not [string]::IsNullOrEmpty($FrontendUrl)) {
    Write-Host "   $FrontendUrl"
} else {
    Write-Host "   (Frontend URL not available)"
}
Write-Host ""
Write-Host "3. Check logs:"
$logCmd = "gcloud run services logs read explainiq-orchestrator --region"
Write-Host "   $logCmd $Region"
Write-Host ""

