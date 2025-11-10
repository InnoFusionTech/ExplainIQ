# Google Cloud Helper Functions
# Suppresses informational stderr messages from gcloud

function Set-GCloudProject {
    param(
        [Parameter(Mandatory=$true)]
        [string]$ProjectId
    )
    
    $ErrorActionPreference = "SilentlyContinue"
    $null = gcloud config set project $ProjectId 2>&1 | Out-Null
    $ErrorActionPreference = "Stop"
    
    $currentProject = gcloud config get-value project 2>$null
    if ($currentProject -eq $ProjectId) {
        Write-Host "✓ Project set to: $ProjectId" -ForegroundColor Green
    } else {
        Write-Host "✗ Failed to set project" -ForegroundColor Red
    }
}

function Get-GCloudProject {
    $project = gcloud config get-value project 2>$null
    $config = gcloud config configurations list --filter="IS_ACTIVE=True" --format="value(name)" 2>$null
    
    Write-Host "Active Configuration: $config" -ForegroundColor Cyan
    Write-Host "Active Project: $project" -ForegroundColor Cyan
    return $project
}

function Switch-GCloudConfig {
    param(
        [Parameter(Mandatory=$true)]
        [string]$ConfigName
    )
    
    $ErrorActionPreference = "SilentlyContinue"
    $null = gcloud config configurations activate $ConfigName 2>&1 | Out-Null
    $ErrorActionPreference = "Stop"
    
    $active = gcloud config configurations list --filter="IS_ACTIVE=True" --format="value(name)" 2>$null
    if ($active -eq $ConfigName) {
        Write-Host "✓ Switched to configuration: $ConfigName" -ForegroundColor Green
        Get-GCloudProject
    } else {
        Write-Host "✗ Failed to switch configuration" -ForegroundColor Red
    }
}

# Export functions
Export-ModuleMember -Function Set-GCloudProject, Get-GCloudProject, Switch-GCloudConfig

