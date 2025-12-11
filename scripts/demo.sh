#!/bin/bash

set -e

echo "=================================================="
echo "Temporal Order Processing - Demo Script"
echo "=================================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if services are running
echo -e "${BLUE}Checking if services are running...${NC}"
if ! docker-compose ps | grep -q "Up"; then
    echo -e "${YELLOW}Services not running. Starting them now...${NC}"
    docker-compose up -d
    echo "Waiting 30 seconds for services to initialize..."
    sleep 30
else
    echo -e "${GREEN}Services are already running!${NC}"
fi

echo ""
echo -e "${BLUE}Service Status:${NC}"
docker-compose ps
echo ""

# Test WireMock
echo -e "${BLUE}Testing WireMock...${NC}"
if curl -s http://localhost:8081/__admin/ > /dev/null; then
    echo -e "${GREEN}✓ WireMock is responding${NC}"
else
    echo -e "${YELLOW}⚠ WireMock might not be ready yet${NC}"
fi
echo ""

echo -e "${GREEN}=================================================="
echo "Demo Ready!"
echo "=================================================="
echo ""
echo "Temporal UI: http://localhost:8080"
echo "WireMock: http://localhost:8081"
echo ""
echo "To run the demo:"
echo "1. Start worker: ${YELLOW}go run worker/main.go${NC}"
echo "2. Start workflow: ${YELLOW}go run starter/main.go${NC}"
echo ""
echo "Or use the Makefile:"
echo "  ${YELLOW}make worker${NC}  - Start the worker"
echo "  ${YELLOW}make start${NC}   - Start a demo workflow"
echo "  ${YELLOW}make test${NC}    - Run tests"
echo ""
echo "=================================================="
