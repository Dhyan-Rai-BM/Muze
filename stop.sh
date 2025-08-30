#!/bin/bash

echo "Stopping Muze Real-Time Post Service"
echo "======================================"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Stop Go services
echo -e "\n${BLUE}Stopping Go services...${NC}"
pkill -f "go run" 2>/dev/null || true
echo -e "${GREEN}Go services stopped${NC}"

# Stop Docker services
echo -e "\n${BLUE}Stopping Docker services...${NC}"
docker-compose down 2>/dev/null || true
echo -e "${GREEN}Docker services stopped${NC}"

# Clean up log files
echo -e "\n${BLUE}Cleaning up log files...${NC}"
rm -f grpc.log graphql.log 2>/dev/null || true
echo -e "${GREEN}Log files cleaned${NC}"

echo -e "\n${GREEN}All services stopped successfully!${NC}"
echo -e "${YELLOW}To start again, run: ./start.sh${NC}"
