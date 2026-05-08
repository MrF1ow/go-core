#!/bin/bash

echo "Starting Authentication API in Development Mode with Hot Reload..."
echo ""
echo "Services will be available at:"
echo "- Auth API: http://localhost:8080"
echo "- Redis Commander: http://localhost:8081"
echo "- PostgreSQL: localhost:5433"
echo ""

docker network inspect shared-api-network >/dev/null 2>&1 || docker network create shared-api-network

docker-compose -f docker-compose.yml -f docker-compose.dev.yml up --build