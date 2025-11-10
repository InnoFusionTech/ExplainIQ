#!/bin/bash
# Convenience script for Docker Compose commands
# Usage: ./docker-compose.sh [frontend|backend|agents|full] [up|down|build|logs]

set -e

COMPOSE_FILE="docker-compose.yml"
COMPOSE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$COMPOSE_DIR"

case "$1" in
  frontend)
    shift
    docker-compose --profile frontend "$@"
    ;;
  backend)
    shift
    docker-compose --profile backend "$@"
    ;;
  agents)
    shift
    docker-compose --profile agents "$@"
    ;;
  full|"")
    docker-compose "$@"
    ;;
  *)
    echo "Usage: $0 [frontend|backend|agents|full] [docker-compose-command]"
    echo ""
    echo "Examples:"
    echo "  $0 frontend up --build"
    echo "  $0 backend up"
    echo "  $0 agents build"
    echo "  $0 full down"
    exit 1
    ;;
esac

