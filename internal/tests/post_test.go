package tests

import (
	"context"
	"muze/internal/cache"
	"muze/internal/database"
	"muze/internal/grpc"
	"muze/internal/messaging"
	"muze/internal/models"
	pb "muze/proto"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostService_CreatePost(t *testing.T) {
	// Skip if no database connection
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test - no database connection configured")
	}

	// Setup
	database.InitDB()
	cache.InitRedis()
	messaging.InitNATS()

	server := grpc.NewPostServer()
	ctx := context.Background()

	// Test data
	req := &pb.CreatePostRequest{
		Content:  "Test post content",
		AuthorId: "test-user-123",
	}

	// Execute
	resp, err := server.CreatePost(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Id)
	assert.Equal(t, req.Content, resp.Content)
	assert.Equal(t, req.AuthorId, resp.AuthorId)
	assert.Equal(t, int32(0), resp.Likes)
	assert.NotEmpty(t, resp.Timestamp)
}

func TestPostService_GetPosts(t *testing.T) {
	// Skip if no database connection
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test - no database connection configured")
	}

	// Setup
	database.InitDB()
	cache.InitRedis()
	messaging.InitNATS()

	server := grpc.NewPostServer()
	ctx := context.Background()

	// Create test posts
	post1 := &pb.CreatePostRequest{
		Content:  "First test post",
		AuthorId: "user1",
	}
	post2 := &pb.CreatePostRequest{
		Content:  "Second test post",
		AuthorId: "user2",
	}

	server.CreatePost(ctx, post1)
	server.CreatePost(ctx, post2)

	// Test get posts
	req := &pb.GetPostsRequest{
		Limit:  10,
		Offset: 0,
	}

	resp, err := server.GetPosts(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(resp.Posts), 2)
	assert.GreaterOrEqual(t, resp.Total, int32(2))
}

func TestPostService_LikePost(t *testing.T) {
	// Skip if no database connection
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping test - no database connection configured")
	}

	// Setup
	database.InitDB()
	cache.InitRedis()
	messaging.InitNATS()

	server := grpc.NewPostServer()
	ctx := context.Background()

	// Create a post first
	createReq := &pb.CreatePostRequest{
		Content:  "Post to like",
		AuthorId: "user1",
	}
	post, err := server.CreatePost(ctx, createReq)
	require.NoError(t, err)

	// Like the post
	likeReq := &pb.LikePostRequest{
		PostId: post.Id,
		UserId: "user2",
	}

	likedPost, err := server.LikePost(ctx, likeReq)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int32(1), likedPost.Likes)
	assert.Equal(t, post.Id, likedPost.Id)
}

func TestCache_RecentPosts(t *testing.T) {
	// Skip if no Redis connection
	if os.Getenv("REDIS_HOST") == "" {
		t.Skip("Skipping test - no Redis connection configured")
	}

	// Setup
	cache.InitRedis()

	// Test data
	posts := []models.Post{
		{
			ID:         "1",
			Content:    "First post",
			AuthorID:   "user1",
			AuthorName: "User 1",
			Likes:      5,
			CreatedAt:  time.Now(),
		},
		{
			ID:         "2",
			Content:    "Second post",
			AuthorID:   "user2",
			AuthorName: "User 2",
			Likes:      10,
			CreatedAt:  time.Now(),
		},
	}

	// Test cache
	err := cache.CacheRecentPosts(posts)
	require.NoError(t, err)

	// Retrieve from cache
	cachedPosts, err := cache.GetRecentPosts()
	require.NoError(t, err)
	assert.Len(t, cachedPosts, 2)
	assert.Equal(t, posts[0].ID, cachedPosts[0].ID)
	assert.Equal(t, posts[1].ID, cachedPosts[1].ID)
}

func TestNATS_Messaging(t *testing.T) {
	// Skip if no NATS connection
	if os.Getenv("NATS_HOST") == "" {
		t.Skip("Skipping test - no NATS connection configured")
	}

	// Setup
	messaging.InitNATS()

	// Test post
	post := models.Post{
		ID:         "test-post",
		Content:    "Test content",
		AuthorID:   "test-user",
		AuthorName: "Test User",
		Likes:      0,
		CreatedAt:  time.Now(),
	}

	// Test publishing
	err := messaging.PublishPostCreated(post)
	require.NoError(t, err)

	// Test subscribing
	received := make(chan bool, 1)
	subscription, err := messaging.SubscribeToPosts(func(data []byte) {
		received <- true
	})
	require.NoError(t, err)
	defer subscription.Unsubscribe()

	// Publish another message
	err = messaging.PublishPostCreated(post)
	require.NoError(t, err)

	// Wait for message
	select {
	case <-received:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for NATS message")
	}
}
