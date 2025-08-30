package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"muze/internal/models"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	ctx := context.Background()
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	log.Println("Redis connected successfully")
}

// CacheRecentPosts caches exactly 10 most recent posts with 5-minute TTL
func CacheRecentPosts(posts []models.Post) error {
	ctx := context.Background()

	// Convert posts to JSON
	postsJSON, err := json.Marshal(posts)
	if err != nil {
		return err
	}

	// Cache with 5-minute TTL as specified
	err = RedisClient.Set(ctx, "recent_posts", postsJSON, 5*time.Minute).Err()
	if err != nil {
		return err
	}

	return nil
}

// GetRecentPosts retrieves cached recent posts
func GetRecentPosts() ([]models.Post, error) {
	ctx := context.Background()

	result, err := RedisClient.Get(ctx, "recent_posts").Result()
	if err != nil {
		return nil, err
	}

	var posts []models.Post
	err = json.Unmarshal([]byte(result), &posts)
	if err != nil {
		return nil, err
	}

	return posts, nil
}

// CachePost caches individual post
func CachePost(post models.Post) error {
	ctx := context.Background()

	postJSON, err := json.Marshal(post)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("post:%s", post.ID)
	err = RedisClient.Set(ctx, key, postJSON, 5*time.Minute).Err()
	if err != nil {
		return err
	}

	return nil
}

// GetCachedPost retrieves cached individual post
func GetCachedPost(postID string) (*models.Post, error) {
	ctx := context.Background()

	key := fmt.Sprintf("post:%s", postID)
	result, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var post models.Post
	err = json.Unmarshal([]byte(result), &post)
	if err != nil {
		return nil, err
	}

	return &post, nil
}

// InvalidatePostCache removes post from cache
func InvalidatePostCache(postID string) error {
	ctx := context.Background()

	key := fmt.Sprintf("post:%s", postID)
	err := RedisClient.Del(ctx, key).Err()
	if err != nil {
		return err
	}

	// Also invalidate recent posts cache
	err = RedisClient.Del(ctx, "recent_posts").Err()
	if err != nil {
		return err
	}

	return nil
}
