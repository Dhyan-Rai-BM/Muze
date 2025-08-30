#!/bin/bash

echo "🚀 Testing Muze Real-Time Post Service"
echo "======================================"

# Check if services are running
echo "📡 Checking service health..."

# Check GraphQL service
echo "Testing GraphQL service..."
curl -s http://localhost:7000/health || echo "❌ GraphQL service not running"

# Check gRPC service (using grpcurl if available)
echo "Testing gRPC service..."
if command -v grpcurl &> /dev/null; then
    grpcurl -plaintext localhost:7001 list || echo "❌ gRPC service not running"
else
    echo "⚠️  grpcurl not installed, skipping gRPC test"
fi

# Test GraphQL queries
echo ""
echo "🧪 Testing GraphQL queries..."

# Test getPosts query
echo "Testing getPosts query..."
curl -s -X POST http://localhost:7000/query \
  -H "Content-Type: application/json" \
  -d '{"query": "query { getPosts(limit: 10, offset: 0) { posts { id content author { name } likes timestamp } total } }"}' \
  | jq '.' || echo "❌ getPosts query failed"

# Test createPost mutation (without auth - should fail)
echo ""
echo "Testing createPost mutation (should fail without auth)..."
curl -s -X POST http://localhost:7000/query \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation { createPost(content: \"Test post\") { id content } }"}' \
  | jq '.' || echo "❌ createPost query failed"

echo ""
echo "✅ Test completed!"
echo ""
echo "To run the complete system:"
echo "1. Start infrastructure: make docker-up"
echo "2. Start gRPC service: make run-grpc"
echo "3. Start GraphQL service: make run-graphql"
echo "4. Run load tests: make load-test"
