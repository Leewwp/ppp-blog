#!/bin/bash

set -euo pipefail

VALIDATION_PATH="${VALIDATION_PATH:-/opt/halo-validation}"
mkdir -p "$VALIDATION_PATH"

cd "$VALIDATION_PATH"

if [ ! -f "validation/server/docker-compose.validation.yml" ]; then
  echo "Validation stack files not found in $VALIDATION_PATH"
  echo "Copy the repository files to $VALIDATION_PATH before running this script."
  exit 1
fi

if [ ! -f "validation/server/.env.validation" ]; then
  cp validation/server/.env.validation.example validation/server/.env.validation
  echo "Created validation/server/.env.validation from example. Fill real secrets before starting the stack."
fi

mkdir -p validation/build/test-results

docker compose -f validation/server/docker-compose.validation.yml \
  --env-file validation/server/.env.validation \
  up -d --remove-orphans

echo "Validation instance prepared at $VALIDATION_PATH"
echo "Open firewall/security-group port 18090 if GitHub Actions must reach it from outside the server."
echo "Expected URL: http://<server-ip>:18090"
