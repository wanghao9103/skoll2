#!/usr/bin/env bash
set -euo pipefail

ACTION="${1:-up}"
COMPOSE_FILE="docker-compose.yml"

run_compose() {
  docker compose -f "${COMPOSE_FILE}" "$@"
}

case "${ACTION}" in
  build)
    run_compose build --pull
    ;;
  up)
    run_compose up -d --build
    ;;
  down)
    run_compose down
    ;;
  restart)
    run_compose down
    run_compose up -d --build
    ;;
  logs)
    run_compose logs -f --tail 200
    ;;
  ps)
    run_compose ps
    ;;
  *)
    echo "Usage: ./scripts/deploy.sh [build|up|down|restart|logs|ps]"
    exit 1
    ;;
esac
