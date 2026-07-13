#!/bin/sh
set -e

echo "Starting BimaNyaya Entrypoint Script..."

# Set migration directories
export MIGRATIONS_DIR="/app/db/migrations"

echo "Applying database migrations..."
/app/migrate

echo "Starting Go API Server..."
exec /app/api
