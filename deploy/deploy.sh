#!/bin/bash

###############################################################################
# Google Cloud Run Deployment Script for ExplainIQ
# 
# This script deploys all ExplainIQ services to Google Cloud Run.
# It handles:
# - Project setup and API enabling
# - Artifact Registry creation
# - Docker image building and pushing
# - Cloud Run service deployment
# - Service-to-service authentication
# - Environment variable configuration
#
# Usage:
#   ./deploy.sh --project-id PROJECT_ID [OPTIONS]
#
# Options:
#   --project-id ID     Google Cloud project ID (required)
#   --region REGION     GCP region (default: europe-west1)
#   --repo REPO         Artifact Registry repository (default: explainiq-repo)
#   --skip-build        Skip Docker build step
#   --dry-run           Show what would be done without executing
#   -h, --help          Show this help message
###############################################################################

set -euo pipefail  # Exit on error, undefined vars, pipe failures

###############################################################################
# Configuration and Constants
###############################################################################

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m' # No Color

# Default values
PROJECT_ID=""
REGION="europe-west1"
REPO="explainiq-repo"
SKIP_BUILD=false
DRY_RUN=false

# Service definitions
readonly SERVICES=(
  "orchestrator:8080:true"
  "summarizer:8081:false"
  "explainer:8082:false"
  "critic:8083:false"
  "visualizer:8084:false"
  "frontend:3000:true"
)

###############################################################################
# Utility Functions
###############################################################################

log_info() {
  echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
  echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
  echo -e "${YELLOW}[WARNING]${NC} $*" >&2
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $*" >&2
}

log_step() {
  echo ""
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${CYAN}▶ $*${NC}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

check_command() {
  if ! command -v "$1" &> /dev/null; then
    log_error "$1 is not installed or not in PATH"
    exit 1
  fi
}

check_gcloud_auth() {
  if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q .; then
    log_error "Not authenticated to gcloud. Run: gcloud auth login"
    exit 1
  fi
}

check_docker() {
  if ! docker info &> /dev/null; then
    log_error "Docker is not running or not accessible"
    exit 1
  fi
}

###############################################################################
# Cleanup Functions
###############################################################################

TMP_DIR=""
cleanup() {
  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    log_info "Cleaning up temporary directory..."
    rm -rf "$TMP_DIR"
  fi
}

trap cleanup EXIT INT TERM

###############################################################################
# Argument Parsing
###############################################################################

show_help() {
  cat << EOF
Usage: $0 [OPTIONS]

Deploy ExplainIQ services to Google Cloud Run

Required:
  --project-id ID     Google Cloud project ID

Options:
  --region REGION     GCP region (default: europe-west1)
  --repo REPO         Artifact Registry repository (default: explainiq-repo)
  --skip-build         Skip Docker build and push step
  --dry-run           Show what would be done without executing
  -h, --help          Show this help message

Examples:
  $0 --project-id my-project-id
  $0 --project-id my-project-id --region us-central1
  $0 --project-id my-project-id --skip-build
  $0 --project-id my-project-id --dry-run
EOF
}

parse_args() {
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
      --dry-run)
        DRY_RUN=true
        shift
        ;;
      -h|--help)
        show_help
        exit 0
        ;;
      *)
        log_error "Unknown option: $1"
        show_help
        exit 1
        ;;
    esac
  done

  if [ -z "$PROJECT_ID" ]; then
    log_error "--project-id is required"
    show_help
    exit 1
  fi
}

###############################################################################
# Setup Functions
###############################################################################

setup_project() {
  log_step "Setting up GCP project"
  
  if [ "$DRY_RUN" = true ]; then
    log_info "Would set project to: $PROJECT_ID"
    return 0
  fi

  log_info "Setting project to: $PROJECT_ID"
  gcloud config set project "$PROJECT_ID" --quiet
  log_success "Project set to $PROJECT_ID"
}

enable_apis() {
  log_step "Enabling required APIs"
  
  local apis=(
    "run.googleapis.com"
    "cloudbuild.googleapis.com"
    "artifactregistry.googleapis.com"
    "secretmanager.googleapis.com"
    "storage.googleapis.com"
  )

  if [ "$DRY_RUN" = true ]; then
    log_info "Would enable APIs: ${apis[*]}"
    return 0
  fi

  log_info "Enabling APIs..."
  for api in "${apis[@]}"; do
    if gcloud services list --enabled --filter="name:$api" --format="value(name)" | grep -q "$api"; then
      log_info "API $api is already enabled"
    else
      log_info "Enabling $api..."
      gcloud services enable "$api" --quiet
      log_success "Enabled $api"
    fi
  done
}

setup_artifact_registry() {
  log_step "Setting up Artifact Registry"
  
  local image_base="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO"
  
  if [ "$DRY_RUN" = true ]; then
    log_info "Would create repository: $REPO in region: $REGION"
    log_info "Image base would be: $image_base"
    return 0
  fi

  if gcloud artifacts repositories describe "$REPO" --location="$REGION" &>/dev/null; then
    log_info "Repository $REPO already exists"
  else
    log_info "Creating repository $REPO in region $REGION..."
    gcloud artifacts repositories create "$REPO" \
      --repository-format=docker \
      --location="$REGION" \
      --description="ExplainIQ Docker images" \
      --quiet
    log_success "Repository $REPO created"
  fi

  log_info "Configuring Docker authentication..."
  gcloud auth configure-docker "$REGION-docker.pkg.dev" --quiet
  log_success "Docker authentication configured"
}

###############################################################################
# Build Functions
###############################################################################

build_and_push_images() {
  if [ "$SKIP_BUILD" = true ]; then
    log_info "Skipping Docker build step (--skip-build specified)"
    return 0
  fi

  log_step "Building and pushing Docker images"
  
  local image_base="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO"
  
  if [ "$DRY_RUN" = true ]; then
    log_info "Would build and push images to: $image_base"
    return 0
  fi

  cd "$PROJECT_ROOT"

  # Build orchestrator
  log_info "Building orchestrator..."
  docker build -f docker/Dockerfile.orchestrator \
    -t "$image_base/explainiq-orchestrator:latest" .
  docker push "$image_base/explainiq-orchestrator:latest"
  log_success "Orchestrator image built and pushed"

  # Build agents
  for service_info in "${SERVICES[@]}"; do
    IFS=':' read -r service_name port is_public <<< "$service_info"
    
    if [ "$service_name" = "orchestrator" ] || [ "$service_name" = "frontend" ]; then
      continue  # Already handled
    fi

    log_info "Building agent-$service_name..."
    docker build -f "docker/Dockerfile.agent-$service_name" \
      -t "$image_base/explainiq-$service_name:latest" .
    docker push "$image_base/explainiq-$service_name:latest"
    log_success "Agent $service_name image built and pushed"
  done

  # Build frontend
  log_info "Building frontend..."
  docker build -f cmd/frontend/nextjs/Dockerfile \
    -t "$image_base/explainiq-frontend:latest" \
    cmd/frontend/nextjs
  docker push "$image_base/explainiq-frontend:latest"
  log_success "Frontend image built and pushed"

  log_success "All images built and pushed successfully"
}

###############################################################################
# Deployment Functions
###############################################################################

prepare_yaml_files() {
  log_step "Preparing Cloud Run YAML files"
  
  local image_base="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO"
  
  TMP_DIR=$(mktemp -d)
  cp "$SCRIPT_DIR/cloudrun"/*.yaml "$TMP_DIR/"

  log_info "Replacing placeholders in YAML files..."
  for yaml in "$TMP_DIR"/*.yaml; do
    # Replace full image path first
    sed -i.bak "s|REGION-docker.pkg.dev/PROJECT_ID/REPO|$image_base|g" "$yaml"
    # Then replace remaining placeholders
    sed -i.bak "s|PROJECT_ID|$PROJECT_ID|g" "$yaml"
    sed -i.bak "s|REGION|$REGION|g" "$yaml"
    # Only replace standalone REPO (not in image paths)
    sed -i.bak "s|\bREPO\b|$REPO|g" "$yaml"
    rm -f "${yaml}.bak"
  done

  log_success "YAML files prepared"
}

deploy_service() {
  local service_name=$1
  local yaml_file=$2
  local allow_public=$3
  
  if [ "$DRY_RUN" = true ]; then
    log_info "Would deploy $service_name (public: $allow_public)"
    return 0
  fi

  log_info "Deploying $service_name..."
  
  local deploy_cmd=(
    gcloud run services replace "$yaml_file"
    --region="$REGION"
    --platform=managed
    --quiet
  )
  
  if [ "$allow_public" = "false" ]; then
    deploy_cmd+=(--no-allow-unauthenticated)
  fi
  
  if "${deploy_cmd[@]}" 2>&1; then
    log_success "$service_name deployed successfully"
    return 0
  else
    log_error "Failed to deploy $service_name"
    return 1
  fi
}

deploy_services() {
  log_step "Deploying Cloud Run services"
  
  prepare_yaml_files

  # Deploy orchestrator first (critical service)
  if ! deploy_service "orchestrator" "$TMP_DIR/orchestrator.yaml" "true"; then
    log_error "Failed to deploy orchestrator. Aborting deployment."
    exit 1
  fi

  # Deploy agents (internal)
  for service_info in "${SERVICES[@]}"; do
    IFS=':' read -r service_name port is_public <<< "$service_info"
    
    if [ "$service_name" = "orchestrator" ] || [ "$service_name" = "frontend" ]; then
      continue  # Will handle separately
    fi

    if ! deploy_service "$service_name" "$TMP_DIR/$service_name.yaml" "$is_public"; then
      log_warning "Failed to deploy $service_name. Continuing with other services..."
    fi
  done

  # Deploy frontend (public)
  if ! deploy_service "frontend" "$TMP_DIR/frontend.yaml" "true"; then
    log_warning "Failed to deploy frontend. Continuing..."
  fi
}

###############################################################################
# Configuration Functions
###############################################################################

get_service_url() {
  local service_name=$1
  local url
  
  url=$(gcloud run services describe "$service_name" \
    --region=$REGION \
    --format='value(status.url)' 2>/dev/null)" || true
  
  if [ -z "$url" ]; then
    log_warning "Could not get URL for $service_name"
    echo ""
  else
    echo "$url"
  fi
}

get_service_urls() {
  log_step "Getting service URLs"
  
  ORCHESTRATOR_URL=$(get_service_url "explainiq-orchestrator")
  SUMMARIZER_URL=$(get_service_url "explainiq-summarizer")
  EXPLAINER_URL=$(get_service_url "explainiq-explainer")
  CRITIC_URL=$(get_service_url "explainiq-critic")
  VISUALIZER_URL=$(get_service_url "explainiq-visualizer")
  FRONTEND_URL=$(get_service_url "explainiq-frontend")

  if [ -z "$ORCHESTRATOR_URL" ]; then
    log_error "Orchestrator service is not deployed or not accessible"
    exit 1
  fi

  if [ -z "$SUMMARIZER_URL" ] || [ -z "$EXPLAINER_URL" ] || \
     [ -z "$CRITIC_URL" ] || [ -z "$VISUALIZER_URL" ]; then
    log_warning "Some agent services are not deployed or not accessible"
  fi
}

configure_authentication() {
  log_step "Configuring service-to-service authentication"
  
  if [ "$DRY_RUN" = true ]; then
    log_info "Would configure IAM bindings for service-to-service auth"
    return 0
  fi

  # Get orchestrator service account
  local orchestrator_sa
  orchestrator_sa=$(gcloud run services describe explainiq-orchestrator \
    --region="$REGION" \
    --format='value(spec.template.spec.serviceAccountName)' 2>/dev/null) || true

  if [ -z "$orchestrator_sa" ]; then
    orchestrator_sa="${PROJECT_ID}-compute@developer.gserviceaccount.com"
    log_info "Using default compute service account: $orchestrator_sa"
  fi

  # Grant orchestrator permission to invoke agents
  local agent_services=("explainiq-summarizer" "explainiq-explainer" "explainiq-critic" "explainiq-visualizer")
  for service in "${agent_services[@]}"; do
    log_info "Granting access to $service..."
    if gcloud run services add-iam-policy-binding "$service" \
      --region="$REGION" \
      --member="serviceAccount:$orchestrator_sa" \
      --role="roles/run.invoker" \
      --quiet 2>&1; then
      log_success "Access granted to $service"
    else
      log_warning "Failed to grant access to $service"
    fi
  done
}

update_environment_variables() {
  log_step "Updating environment variables"
  
  if [ "$DRY_RUN" = true ]; then
    log_info "Would update environment variables for orchestrator and frontend"
    return 0
  fi

  # Update orchestrator with agent URLs
  if [ -n "$SUMMARIZER_URL" ] && [ -n "$EXPLAINER_URL" ] && \
     [ -n "$CRITIC_URL" ] && [ -n "$VISUALIZER_URL" ]; then
    log_info "Updating orchestrator environment variables..."
    if gcloud run services update explainiq-orchestrator \
      --region="$REGION" \
      --update-env-vars="AGENT_SUMMARIZER_URL=$SUMMARIZER_URL,AGENT_EXPLAINER_URL=$EXPLAINER_URL,AGENT_CRITIC_URL=$CRITIC_URL,AGENT_VISUALIZER_URL=$VISUALIZER_URL,GCP_PROJECT_ID=$PROJECT_ID,SERVICE_URL=$ORCHESTRATOR_URL" \
      --quiet 2>&1; then
      log_success "Orchestrator environment variables updated"
    else
      log_warning "Failed to update orchestrator environment variables"
    fi
  else
    log_warning "Skipping orchestrator environment variable update - some agent URLs are missing"
  fi

  # Update frontend with orchestrator URL
  if [ -n "$ORCHESTRATOR_URL" ] && [ -n "$FRONTEND_URL" ]; then
    log_info "Updating frontend environment variables..."
    if gcloud run services update explainiq-frontend \
      --region="$REGION" \
      --update-env-vars="NEXT_PUBLIC_ORCHESTRATOR_URL=$ORCHESTRATOR_URL,GCS_PROJECT_ID=$PROJECT_ID" \
      --quiet 2>&1; then
      log_success "Frontend environment variables updated"
    else
      log_warning "Failed to update frontend environment variables"
    fi
  else
    log_warning "Skipping frontend environment variable update - URLs are missing"
  fi
}

###############################################################################
# Summary Functions
###############################################################################

show_summary() {
  log_step "Deployment Summary"
  
  echo ""
  echo -e "${GREEN}════════════════════════════════════════════════${NC}"
  echo -e "${GREEN}  Deployment Complete!${NC}"
  echo -e "${GREEN}════════════════════════════════════════════════${NC}"
  echo ""
  echo -e "${CYAN}Service URLs:${NC}"
  echo "  Orchestrator: ${ORCHESTRATOR_URL:-N/A}"
  echo "  Frontend:     ${FRONTEND_URL:-N/A}"
  echo ""
  echo -e "${CYAN}Agent URLs (internal):${NC}"
  echo "  Summarizer:  ${SUMMARIZER_URL:-N/A}"
  echo "  Explainer:   ${EXPLAINER_URL:-N/A}"
  echo "  Critic:      ${CRITIC_URL:-N/A}"
  echo "  Visualizer:  ${VISUALIZER_URL:-N/A}"
  echo ""
  echo -e "${YELLOW}Next Steps:${NC}"
  echo "1. Verify services are running:"
  echo "   ${CYAN}gcloud run services list --region=$REGION${NC}"
  echo ""
  echo "2. Test the frontend:"
  if [ -n "$FRONTEND_URL" ]; then
    echo "   ${CYAN}$FRONTEND_URL${NC}"
  else
    echo "   ${RED}Frontend URL not available${NC}"
  fi
  echo ""
  echo "3. Check logs:"
  echo "   ${CYAN}gcloud run services logs read explainiq-orchestrator --region=$REGION${NC}"
  echo ""
}

###############################################################################
# Main Function
###############################################################################

main() {
  echo ""
  echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║   ExplainIQ Cloud Run Deployment Script       ║${NC}"
  echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
  echo ""
  echo "Project ID: $PROJECT_ID"
  echo "Region:     $REGION"
  echo "Repository: $REPO"
  echo "Skip Build: $SKIP_BUILD"
  echo "Dry Run:    $DRY_RUN"
  echo ""

  # Pre-flight checks
  check_command gcloud
  check_command docker
  check_gcloud_auth
  check_docker

  # Setup
  setup_project
  enable_apis
  setup_artifact_registry

  # Build
  build_and_push_images

  # Deploy
  deploy_services

  # Configure
  get_service_urls
  configure_authentication
  update_environment_variables

  # Summary
  show_summary
}

###############################################################################
# Script Entry Point
###############################################################################

parse_args "$@"
main

