#!/bin/bash

# Google Cloud Run Deployment Script
# Deploys all ExplainIQ services to Cloud Run

set -e

# Cleanup function
cleanup() {
  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
PROJECT_ID=""
REGION="europe-west1"
REPO="explainiq-repo"
SKIP_BUILD=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --project-id)
      PROJECT_ID="$2"
      shift 2
      ;;
    --region)
      REGION="$2"
      shift 2
      ;;
    --repo)
      REPO="$2"
      shift 2
      ;;
    --skip-build)
      SKIP_BUILD=true
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --project-id ID    Google Cloud project ID (required)"
      echo "  --region REGION   GCP region (default: europe-west1)"
      echo "  --repo REPO       Artifact Registry repository (default: explainiq-repo)"
      echo "  --skip-build      Skip Docker build step"
      echo "  -h, --help        Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Validate required parameters
if [ -z "$PROJECT_ID" ]; then
  echo -e "${RED}Error: --project-id is required${NC}"
  exit 1
fi

echo -e "${GREEN}Deploying ExplainIQ to Google Cloud Run${NC}"
echo "Project ID: $PROJECT_ID"
echo "Region: $REGION"
echo "Repository: $REPO"
echo ""

# Set project
echo -e "${YELLOW}Setting GCP project...${NC}"
gcloud config set project "$PROJECT_ID"

# Enable required APIs
echo -e "${YELLOW}Enabling required APIs...${NC}"
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  storage.googleapis.com \
  --quiet

# Create Artifact Registry repository if it doesn't exist
echo -e "${YELLOW}Setting up Artifact Registry...${NC}"
if ! gcloud artifacts repositories describe "$REPO" --location="$REGION" &>/dev/null; then
  gcloud artifacts repositories create "$REPO" \
    --repository-format=docker \
    --location="$REGION" \
    --description="ExplainIQ Docker images"
fi

# Configure Docker authentication
gcloud auth configure-docker "$REGION-docker.pkg.dev" --quiet

# Build and push images
if [ "$SKIP_BUILD" = false ]; then
  echo -e "${YELLOW}Building and pushing Docker images...${NC}"
  
  IMAGE_BASE="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO"
  
  # Build orchestrator
  echo "Building orchestrator..."
  docker build -f docker/Dockerfile.orchestrator -t "$IMAGE_BASE/explainiq-orchestrator:latest" .
  docker push "$IMAGE_BASE/explainiq-orchestrator:latest"
  
  # Build agents
  echo "Building agent-summarizer..."
  docker build -f docker/Dockerfile.agent-summarizer -t "$IMAGE_BASE/explainiq-summarizer:latest" .
  docker push "$IMAGE_BASE/explainiq-summarizer:latest"
  
  echo "Building agent-explainer..."
  docker build -f docker/Dockerfile.agent-explainer -t "$IMAGE_BASE/explainiq-explainer:latest" .
  docker push "$IMAGE_BASE/explainiq-explainer:latest"
  
  echo "Building agent-critic..."
  docker build -f docker/Dockerfile.agent-critic -t "$IMAGE_BASE/explainiq-critic:latest" .
  docker push "$IMAGE_BASE/explainiq-critic:latest"
  
  echo "Building agent-visualizer..."
  docker build -f docker/Dockerfile.agent-visualizer -t "$IMAGE_BASE/explainiq-visualizer:latest" .
  docker push "$IMAGE_BASE/explainiq-visualizer:latest"
  
  echo "Building frontend..."
  docker build -f cmd/frontend/nextjs/Dockerfile -t "$IMAGE_BASE/explainiq-frontend:latest" cmd/frontend/nextjs
  docker push "$IMAGE_BASE/explainiq-frontend:latest"
  
  echo -e "${GREEN}All images built and pushed successfully${NC}"
fi

# Update YAML files with project ID and image paths
echo -e "${YELLOW}Updating Cloud Run YAML files...${NC}"
IMAGE_BASE="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO"

# Create temporary directory for updated YAMLs
TMP_DIR=$(mktemp -d)
cp deploy/cloudrun/*.yaml "$TMP_DIR/"

# Replace placeholders in all YAML files
# IMPORTANT: Replace full path first, then individual placeholders
# This prevents double replacement of REPO
for yaml in "$TMP_DIR"/*.yaml; do
  # Replace full image path first (this replaces REGION, PROJECT_ID, and REPO in the path)
  sed -i.bak "s|REGION-docker.pkg.dev/PROJECT_ID/REPO|$IMAGE_BASE|g" "$yaml"
  # Then replace remaining standalone placeholders (not in image paths)
  sed -i.bak "s|PROJECT_ID|$PROJECT_ID|g" "$yaml"
  sed -i.bak "s|REGION|$REGION|g" "$yaml"
  # Only replace standalone REPO (not part of the image path)
  # Use word boundaries to avoid replacing REPO in explainiq-repo
  sed -i.bak "s|\bREPO\b|$REPO|g" "$yaml"
  rm -f "$yaml.bak"
done

# Deploy services
echo -e "${YELLOW}Deploying Cloud Run services...${NC}"

# Function to deploy a service with error handling
deploy_service() {
  local service_name=$1
  local yaml_file=$2
  local allow_public=$3
  
  echo "Deploying $service_name..."
  
  local deploy_cmd="gcloud run services replace \"$yaml_file\" --region=\"$REGION\" --platform=managed --quiet"
  
  if [ "$allow_public" = "false" ]; then
    deploy_cmd="$deploy_cmd --no-allow-unauthenticated"
  fi
  
  if ! eval "$deploy_cmd"; then
    echo -e "${RED}ERROR: Failed to deploy $service_name${NC}"
    return 1
  fi
  
  echo -e "${GREEN}✓ $service_name deployed successfully${NC}"
  return 0
}

# Deploy orchestrator (public)
if ! deploy_service "orchestrator" "$TMP_DIR/orchestrator.yaml" "true"; then
  echo -e "${RED}Failed to deploy orchestrator. Aborting deployment.${NC}"
  exit 1
fi

# Deploy agents (internal - no public ingress)
deploy_service "agent-summarizer" "$TMP_DIR/summarizer.yaml" "false" || echo -e "${YELLOW}Warning: Failed to deploy summarizer. Continuing...${NC}"
deploy_service "agent-explainer" "$TMP_DIR/explainer.yaml" "false" || echo -e "${YELLOW}Warning: Failed to deploy explainer. Continuing...${NC}"
deploy_service "agent-critic" "$TMP_DIR/critic.yaml" "false" || echo -e "${YELLOW}Warning: Failed to deploy critic. Continuing...${NC}"
deploy_service "agent-visualizer" "$TMP_DIR/visualizer.yaml" "false" || echo -e "${YELLOW}Warning: Failed to deploy visualizer. Continuing...${NC}"

# Deploy frontend (public)
deploy_service "frontend" "$TMP_DIR/frontend.yaml" "true" || echo -e "${YELLOW}Warning: Failed to deploy frontend. Continuing...${NC}"

# Cleanup temp directory (trap will also handle this on exit)
cleanup

# Get service URLs with error handling
echo -e "${YELLOW}Getting service URLs...${NC}"

get_service_url() {
  local service_name=$1
  local url=$(gcloud run services describe "$service_name" --region="$REGION" --format="value(status.url)" 2>/dev/null)
  
  if [ -z "$url" ]; then
    echo -e "${YELLOW}  Warning: Could not get URL for $service_name${NC}" >&2
    echo ""
  else
    echo "$url"
  fi
}

ORCHESTRATOR_URL=$(get_service_url "explainiq-orchestrator")
SUMMARIZER_URL=$(get_service_url "explainiq-summarizer")
EXPLAINER_URL=$(get_service_url "explainiq-explainer")
CRITIC_URL=$(get_service_url "explainiq-critic")
VISUALIZER_URL=$(get_service_url "explainiq-visualizer")
FRONTEND_URL=$(get_service_url "explainiq-frontend")

# Verify critical services are deployed
if [ -z "$ORCHESTRATOR_URL" ]; then
  echo -e "${RED}ERROR: Orchestrator service is not deployed or not accessible${NC}"
  exit 1
fi

if [ -z "$SUMMARIZER_URL" ] || [ -z "$EXPLAINER_URL" ] || [ -z "$CRITIC_URL" ] || [ -z "$VISUALIZER_URL" ]; then
  echo -e "${YELLOW}WARNING: Some agent services are not deployed or not accessible${NC}"
fi

# Configure service-to-service authentication
echo -e "${YELLOW}Configuring service-to-service authentication...${NC}"

# Get orchestrator service account
ORCHESTRATOR_SA=$(gcloud run services describe explainiq-orchestrator \
  --region="$REGION" \
  --format="value(spec.template.spec.serviceAccountName)" 2>/dev/null)

if [ -z "$ORCHESTRATOR_SA" ]; then
  # Use default compute service account
  ORCHESTRATOR_SA="${PROJECT_ID}-compute@developer.gserviceaccount.com"
  echo -e "${YELLOW}Using default compute service account: $ORCHESTRATOR_SA${NC}"
fi

# Grant orchestrator permission to invoke agents
for service in explainiq-summarizer explainiq-explainer explainiq-critic explainiq-visualizer; do
  echo "Granting access to $service..."
  gcloud run services add-iam-policy-binding "$service" \
    --region="$REGION" \
    --member="serviceAccount:$ORCHESTRATOR_SA" \
    --role="roles/run.invoker" \
    --quiet
done

# Update orchestrator with agent URLs (only if all URLs are available)
if [ -n "$SUMMARIZER_URL" ] && [ -n "$EXPLAINER_URL" ] && [ -n "$CRITIC_URL" ] && [ -n "$VISUALIZER_URL" ]; then
  echo -e "${YELLOW}Updating orchestrator environment variables...${NC}"
  if gcloud run services update explainiq-orchestrator \
    --region="$REGION" \
    --update-env-vars="AGENT_SUMMARIZER_URL=$SUMMARIZER_URL,AGENT_EXPLAINER_URL=$EXPLAINER_URL,AGENT_CRITIC_URL=$CRITIC_URL,AGENT_VISUALIZER_URL=$VISUALIZER_URL,GCP_PROJECT_ID=$PROJECT_ID,SERVICE_URL=$ORCHESTRATOR_URL" \
    --quiet; then
    echo -e "${GREEN}✓ Orchestrator environment variables updated${NC}"
  else
    echo -e "${YELLOW}WARNING: Failed to update orchestrator environment variables${NC}"
  fi
else
  echo -e "${YELLOW}WARNING: Skipping orchestrator environment variable update - some agent URLs are missing${NC}"
fi

# Update frontend with orchestrator URL (only if orchestrator URL is available)
if [ -n "$ORCHESTRATOR_URL" ] && [ -n "$FRONTEND_URL" ]; then
  echo -e "${YELLOW}Updating frontend environment variables...${NC}"
  if gcloud run services update explainiq-frontend \
    --region="$REGION" \
    --update-env-vars="NEXT_PUBLIC_ORCHESTRATOR_URL=$ORCHESTRATOR_URL,GCS_PROJECT_ID=$PROJECT_ID" \
    --quiet; then
    echo -e "${GREEN}✓ Frontend environment variables updated${NC}"
  else
    echo -e "${YELLOW}WARNING: Failed to update frontend environment variables${NC}"
  fi
else
  echo -e "${YELLOW}WARNING: Skipping frontend environment variable update - orchestrator or frontend URL is missing${NC}"
fi

# Display deployment summary
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Service URLs:"
echo "  Orchestrator: $ORCHESTRATOR_URL"
echo "  Frontend:     $FRONTEND_URL"
echo ""
echo "Agent URLs (internal):"
echo "  Summarizer:    $SUMMARIZER_URL"
echo "  Explainer:   $EXPLAINER_URL"
echo "  Critic:      $CRITIC_URL"
echo "  Visualizer:  $VISUALIZER_URL"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "1. Verify services are running:"
echo "   gcloud run services list --region=$REGION"
echo ""
echo "2. Test the frontend:"
echo "   $FRONTEND_URL"
echo ""
echo "3. Check logs:"
echo "   gcloud run services logs read explainiq-orchestrator --region=$REGION"
echo ""

