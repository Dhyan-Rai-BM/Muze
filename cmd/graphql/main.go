package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "muze/proto"
)

type contextKey string

const userContextKey contextKey = "user"

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using system environment variables")
	}

	// Connect to gRPC service
	grpcAddr := os.Getenv("GRPC_HOST")
	if grpcAddr == "" {
		grpcAddr = "localhost:7001"
	}

	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC service: %v", err)
	}
	defer conn.Close()

	grpcClient := pb.NewPostServiceClient(conn)

	// Generate a JWT token for testing
	jwtToken := generateJWT()

	// GraphQL playground HTML
	playgroundHTML := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Muze GraphQL Playground</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .section { margin-bottom: 30px; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        .section h3 { margin-top: 0; color: #555; }
        .code-block { background: #f8f8f8; padding: 15px; border-radius: 5px; font-family: monospace; white-space: pre-wrap; word-break: break-all; }
        .jwt-token { background: #e8f5e8; border: 1px solid #4caf50; color: #2e7d32; padding: 15px; border-radius: 5px; font-family: monospace; font-size: 12px; overflow-wrap: break-word; }
        .url { color: #2196f3; text-decoration: none; }
        .url:hover { text-decoration: underline; }
        .note { background: #fff3cd; border: 1px solid #ffeaa7; color: #856404; padding: 15px; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Muze GraphQL Playground</h1>
        
        <div class="section">
            <h3>JWT Token for Mutations</h3>
            <div class="jwt-token">%s</div>
            <p><strong>Use this token in the Authorization header for mutations:</strong> <code>Bearer %s</code></p>
        </div>

        <div class="section">
            <h3>GraphQL Endpoint</h3>
            <p><strong>URL:</strong> <a href="/query" class="url">http://localhost:7002/query</a></p>
        </div>

        <div class="section">
            <h3>Working cURL Commands</h3>
            <p><strong>Copy and paste these commands into your terminal:</strong></p>
            
            <h4>Get Posts (No Auth Required)</h4>
            <div class="code-block">curl -X POST http://localhost:7002/query -H "Content-Type: application/json" -d '{"query": "query { getPosts(limit: 5) { posts { id content author { name } likes timestamp } } }"}'</div>

            <h4>Create Post (Auth Required)</h4>
            <div class="code-block">curl -X POST http://localhost:7002/query -H "Content-Type: application/json" -H "Authorization: Bearer %s" -d '{"query": "mutation { createPost(content: \"Hello Muze!\") { id content author { name } likes } }"}'</div>

            <h4>Like Post (Auth Required)</h4>
            <div class="code-block">curl -X POST http://localhost:7002/query -H "Content-Type: application/json" -H "Authorization: Bearer %s" -d '{"query": "mutation { likePost(postId: \"1\") { id likes } }"}'</div>
        </div>

        <div class="section">
            <h3>Health Check</h3>
            <p><strong>URL:</strong> <a href="/health" class="url">http://localhost:7002/health</a></p>
            <div class="code-block">curl http://localhost:7002/health</div>
        </div>

        <div class="note">
            <strong>Note:</strong> GET requests (queries) don't require authentication. Only mutations (create, like) require the JWT token.
        </div>
    </div>
</body>
</html>
`, jwtToken, jwtToken, jwtToken, jwtToken)

	// HTTP handler for GraphQL
	http.HandleFunc("/playground", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(playgroundHTML))
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "muze-graphql",
			"grpc":      "connected",
		})
	})

	// GraphQL endpoint
	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Query         string                 `json:"query"`
			Variables     map[string]interface{} `json:"variables"`
			OperationName string                 `json:"operationName"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Check if it's a mutation (requires auth)
		isMutation := strings.Contains(strings.ToLower(req.Query), "mutation")

		var userID string
		if isMutation {
			// Extract JWT token from Authorization header for mutations
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []map[string]interface{}{
						{"message": "Authorization required for mutations"},
					},
				})
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			userID = validateJWT(token)
			if userID == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []map[string]interface{}{
						{"message": "Invalid token"},
					},
				})
				return
			}
		}

		// Create context with user info
		ctx := context.Background()
		if userID != "" {
			ctx = context.WithValue(ctx, userContextKey, userID)
		}

		// Add metadata for gRPC calls
		md := metadata.Pairs("user-id", userID)
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Handle GraphQL queries
		result, err := executeGraphQLQuery(ctx, grpcClient, req.Query, req.Variables)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []map[string]interface{}{
					{"message": err.Error()},
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "7002"
	}

	log.Printf("GraphQL server starting on port %s", port)
	log.Printf("Playground available at: http://localhost:%s/playground", port)
	log.Printf("GraphQL endpoint: http://localhost:%s/query", port)
	log.Printf("Health check: http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func executeGraphQLQuery(ctx context.Context, grpcClient pb.PostServiceClient, query string, variables map[string]interface{}) (map[string]interface{}, error) {
	// Handle GetPosts query (no auth required)
	if strings.Contains(query, "getPosts") {
		limit := int32(10)
		if vars, ok := variables["limit"]; ok {
			if l, ok := vars.(float64); ok {
				limit = int32(l)
			}
		}

		resp, err := grpcClient.GetPosts(ctx, &pb.GetPostsRequest{Limit: limit})
		if err != nil {
			return nil, fmt.Errorf("failed to get posts: %v", err)
		}

		posts := make([]map[string]interface{}, len(resp.Posts))
		for i, post := range resp.Posts {
			posts[i] = convertProtoPostToGraphQL(post)
		}

		return map[string]interface{}{
			"data": map[string]interface{}{
				"getPosts": map[string]interface{}{
					"posts": posts,
				},
			},
		}, nil
	}

	// Handle CreatePost mutation (auth required)
	if strings.Contains(query, "createPost") {
		content := ""
		if vars, ok := variables["content"]; ok {
			if c, ok := vars.(string); ok {
				content = c
			}
		}

		if content == "" {
			// Extract content from the query string
			if strings.Contains(query, "content:") {
				start := strings.Index(query, "content:") + 8
				end := strings.Index(query[start:], "\"")
				if end != -1 {
					content = query[start : start+end]
				}
			}
		}

		if content == "" {
			return nil, fmt.Errorf("content is required")
		}

		userID := getUserIDFromContext(ctx)
		if userID == "" {
			userID = "default-user"
		}

		resp, err := grpcClient.CreatePost(ctx, &pb.CreatePostRequest{
			Content:  content,
			AuthorId: userID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create post: %v", err)
		}

		return map[string]interface{}{
			"data": map[string]interface{}{
				"createPost": convertProtoPostToGraphQL(resp),
			},
		}, nil
	}

	// Handle LikePost mutation (auth required)
	if strings.Contains(query, "likePost") {
		postID := ""
		if vars, ok := variables["postId"]; ok {
			if id, ok := vars.(string); ok {
				postID = id
			}
		}

		if postID == "" {
			// Extract postId from the query string
			if strings.Contains(query, "postId:") {
				start := strings.Index(query, "postId:") + 7
				end := strings.Index(query[start:], "\"")
				if end != -1 {
					postID = query[start : start+end]
				}
			}
		}

		if postID == "" {
			return nil, fmt.Errorf("postId is required")
		}

		userID := getUserIDFromContext(ctx)
		if userID == "" {
			userID = "default-user"
		}

		resp, err := grpcClient.LikePost(ctx, &pb.LikePostRequest{
			PostId: postID,
			UserId: userID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to like post: %v", err)
		}

		return map[string]interface{}{
			"data": map[string]interface{}{
				"likePost": convertProtoPostToGraphQL(resp),
			},
		}, nil
	}

	return nil, fmt.Errorf("unsupported query: %s", query)
}

func convertProtoPostToGraphQL(post *pb.Post) map[string]interface{} {
	var imageURL *string
	if post.ImageUrl != nil {
		imgURL := post.ImageUrl.Value
		imageURL = &imgURL
	}

	return map[string]interface{}{
		"id":        post.Id,
		"content":   post.Content,
		"author":    map[string]interface{}{"name": post.AuthorName},
		"likes":     post.Likes,
		"timestamp": post.Timestamp,
		"imageUrl":  imageURL,
	}
}

func getUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userContextKey).(string); ok {
		return userID
	}
	return ""
}

func generateJWT() string {
	// Simple JWT token generation for testing
	header := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	payload := "eyJzdWIiOiJ0ZXN0LXVzZXIiLCJpYXQiOjE2MzQ1Njc4OTYsImV4cCI6MTYzNDU3MTQ5Nn0"
	signature := "test-signature-for-development"
	
	return header + "." + payload + "." + signature
}

func validateJWT(token string) string {
	// Simple JWT validation for testing
	if strings.Contains(token, "test-signature-for-development") {
		return "test-user"
	}
	return ""
}
