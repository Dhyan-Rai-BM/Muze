# Muze Real-Time Post Service

A complete real-time post service built with Go, GraphQL, and gRPC. Perfect for learning, demos, and interviews.

## What It Does

- GraphQL API for flexible data queries
- gRPC backend for high-performance communication
- Real-time updates with NATS messaging
- PostgreSQL database with Redis caching
- JWT authentication for secure operations
- Docker containerization for easy deployment

## Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose

## Quick Start

### 1. Clone and Run

```bash
git clone <repository-url>
cd muze
./start.sh
```

**./start.sh** The script does everything automatically:
- Sets up environment configuration
- Starts all services (PostgreSQL, Redis, NATS)
- Builds and runs gRPC + GraphQL services
- Generates JWT token for testing
- Opens GraphQL playground at **http://localhost:7002/playground**

**Just click the URL that appears in your terminal!**

### 3. Access Services

- **GraphQL Playground**: http://localhost:7002/playground
- **GraphQL Endpoint**: http://localhost:7002/query
- **gRPC Service**: localhost:7001
- **Health Check**: http://localhost:7002/health

## Configuration

No configuration needed! The service automatically creates a `.env` file with default values.

**All services are pre-configured and ready to run.**

## API Usage

### GraphQL Examples

**Get Posts:**
```graphql
query {
  getPosts(limit: 5) {
    posts {
      id
      content
      author { name }
      likes
      timestamp
    }
  }
}
```

**Create Post:**
```graphql
mutation {
  createPost(content: "Hello Muze!") {
    id
    content
    author { name }
    likes
  }
}
```

**Note:** JWT token is automatically generated and shown in the playground.

## Project Structure

```
muze/
├── start.sh                # One-click startup script
├── stop.sh                 # Stop all services
├── cmd/                    # Application entry points
├── internal/               # Core packages
├── graphql/                # GraphQL schema
├── proto/                  # Protocol Buffers
└── docker-compose.yml      # Infrastructure
```

## Testing

### Manual Testing

**1. GraphQL Testing:**
- Use the playground at http://localhost:7002/playground
- JWT token is automatically generated and displayed
- Test queries and mutations directly in the browser

**2. gRPC Testing:**
- gRPC service runs on port 7001
- Use tools like `grpcurl` or `BloomRPC` for testing
- Example with grpcurl:
```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Test GetPosts
grpcurl -plaintext localhost:7001 list
grpcurl -plaintext localhost:7001 PostService.GetPosts
```

**3. Redis Caching Test:**
- Create a post via GraphQL
- Check Redis logs: `docker logs muze-redis-1`
- Verify caching behavior in the application

## GitHub Actions

The repository includes GitHub Actions workflows for automated testing and deployment. **Note: AWS credentials are currently set to dummy values.**

## Scalability, Security & Performance

**Scalability:** The microservices architecture allows independent scaling of GraphQL and gRPC services. Redis caching reduces database load, while NATS enables horizontal scaling of message processing. PostgreSQL with proper indexing ensures efficient data retrieval.

**Security:** JWT authentication secures all mutations, with tokens validated on every request. Environment variables keep sensitive configuration separate from code. The gRPC service runs internally, exposing only the GraphQL API publicly.

**Performance:** Redis caches the 10 most recent posts with 5-minute TTL, dramatically reducing database queries. gRPC provides high-performance internal communication. Database indexes optimize query performance for large datasets.

## Health Check

- **Health Endpoint**: `GET /health` - Check if services are running
- **Logs**: View service logs with `tail -f grpc.log` or `tail -f graphql.log`
# Muze
