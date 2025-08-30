#!/bin/bash

echo "Starting Muze Post Service"
echo "======================================"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Auto-create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo -e "\n${BLUE}Creating .env file from env.example...${NC}"
    cp env.example .env
    echo -e "${GREEN}.env file created with default values!${NC}"
fi

# Function to check if port is in use
check_port() {
    if lsof -Pi :$1 -sTCP:LISTEN -t >/dev/null ; then
        echo -e "${YELLOW}⚠️  Port $1 is already in use. Stopping existing service...${NC}"
        lsof -ti:$1 | xargs kill -9 2>/dev/null || true
        sleep 2
    fi
}

# Stop any existing services
echo -e "\n${BLUE}Stopping any existing services...${NC}"
pkill -f "go run" 2>/dev/null || true
docker-compose down 2>/dev/null || true

# Check and free up ports
check_port 7001
check_port 7002
check_port 5432
check_port 6380
check_port 4222

# Start infrastructure services
echo -e "\n${BLUE}Starting Docker services...${NC}"
docker-compose up -d

# Wait for services to be ready
echo -e "\n${BLUE}Waiting for services to be ready...${NC}"
sleep 10

# Check if services are running
echo -e "\n${BLUE}Checking service status...${NC}"
if docker ps | grep -q "muze_postgres"; then
    echo -e "${GREEN}PostgreSQL is running${NC}"
else
    echo -e "${YELLOW}PostgreSQL might not be ready yet${NC}"
fi

if docker ps | grep -q "muze_redis"; then
    echo -e "${GREEN}Redis is running${NC}"
else
    echo -e "${YELLOW}Redis might not be ready yet${NC}"
fi

if docker ps | grep -q "muze_nats"; then
    echo -e "${GREEN}NATS is running${NC}"
else
    echo -e "${YELLOW}NATS might not be ready yet${NC}"
fi

# JWT token will be generated inline by the GraphQL service

# Start gRPC service
echo -e "\n${BLUE}Starting gRPC service...${NC}"
go run ./cmd/grpc > grpc.log 2>&1 &
GRPC_PID=$!

# Wait a moment for gRPC to start
sleep 3

# Start GraphQL service
echo -e "\n${BLUE}Starting GraphQL service...${NC}"
go run ./cmd/graphql > graphql.log 2>&1 &
GRAPHQL_PID=$!

# Wait for services to be ready
sleep 5

# Check if services are running
if ps -p $GRPC_PID > /dev/null; then
    echo -e "${GREEN}gRPC service is running on port 7001${NC}"
else
    echo -e "${YELLOW}gRPC service might have issues${NC}"
fi

if ps -p $GRAPHQL_PID > /dev/null; then
    echo -e "${GREEN}GraphQL service is running on port 7002${NC}"
else
    echo -e "${YELLOW}GraphQL service might have issues${NC}"
fi

# Test GraphQL endpoint
echo -e "\n${BLUE}Testing GraphQL endpoint...${NC}"
RESPONSE=$(curl -s -X POST http://localhost:7002/query \
  -H "Content-Type: application/json" \
  -d '{"query": "query { getPosts(limit: 1) { total } }"}')

if echo "$RESPONSE" | grep -q "total"; then
    echo -e "${GREEN}GraphQL API is responding${NC}"
else
    echo -e "${YELLOW}GraphQL API might not be ready yet${NC}"
fi

# Display success message
echo -e "\n${GREEN}Muze Real-Time Post Service is ready!${NC}"
echo "================================================"

echo -e "\n${BLUE}GRAPHQL PLAYGROUND IS READY!${NC}"
echo -e "\n${GREEN}CLICK THIS URL: http://localhost:7002/playground${NC}"
echo -e "${GREEN}CLICK THIS URL: http://localhost:7002/playground${NC}"
echo -e "${GREEN}CLICK THIS URL: http://localhost:7002/playground${NC}"

echo -e "\n${BLUE}Or copy and paste in your browser:${NC}"
echo -e "${YELLOW}http://localhost:7002/playground${NC}"

echo -e "\n${BLUE}JWT Token:${NC}"
echo -e "${YELLOW}Automatically generated and shown in the playground${NC}"

echo -e "\n${BLUE}Quick Commands:${NC}"
echo -e "${YELLOW}Stop services: ./stop.sh${NC}"
echo -e "${YELLOW}Health check: http://localhost:7002/health${NC}"

echo -e "\n${GREEN}STARTUP COMPLETED${NC}"
