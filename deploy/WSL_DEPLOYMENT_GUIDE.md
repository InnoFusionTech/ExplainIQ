# Quick Start Guide for WSL Deployment

## Prerequisites
- WSL installed and running
- gcloud CLI installed in WSL
- Docker installed in WSL
- Authenticated to gcloud: gcloud auth login

## Steps to Deploy

1. Open WSL terminal

2. Navigate to project directory:
   cd /mnt/c/App/ExplainIQ\ Agent

3. Make script executable:
   chmod +x deploy/deploy.sh

4. Run deployment:
   ./deploy/deploy.sh --project-id YOUR_PROJECT_ID

## Alternative: Use existing script
   ./deploy/deploy-cloudrun.sh --project-id YOUR_PROJECT_ID

## Common Options
   --region REGION        (default: europe-west1)
   --repo REPO           (default: explainiq-repo)
   --skip-build          (skip Docker build)
   --dry-run             (test without deploying)

## Example
   ./deploy/deploy.sh --project-id explainiq-477714 --region europe-west1
