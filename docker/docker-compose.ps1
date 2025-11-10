# PowerShell convenience script for Docker Compose commands
# Usage: 
#   .\docker-compose.ps1 -frontend up --build
#   .\docker-compose.ps1 -backend up
#   .\docker-compose.ps1 -agents build
#   .\docker-compose.ps1 up --build  (full stack)

param(
    [Parameter(Mandatory=$false)]
    [switch]$Frontend,
    
    [Parameter(Mandatory=$false)]
    [switch]$Backend,
    
    [Parameter(Mandatory=$false)]
    [switch]$Agents,
    
    [Parameter(Mandatory=$false, ValueFromRemainingArguments=$true)]
    [string[]]$DockerComposeArgs = @("up", "--build")
)

$ComposeDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ComposeDir

$ComposeFile = "docker-compose.yml"

if ($Frontend) {
    docker-compose --profile frontend $DockerComposeArgs
}
elseif ($Backend) {
    docker-compose --profile backend $DockerComposeArgs
}
elseif ($Agents) {
    docker-compose --profile agents $DockerComposeArgs
}
else {
    # Full stack - no profile needed
    docker-compose $DockerComposeArgs
}

