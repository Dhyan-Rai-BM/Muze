package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"muze/internal/models"
	"os"

	"github.com/nats-io/nats.go"
)

var NatsClient *nats.Conn

func InitNATS() {
	var err error
	NatsClient, err = nats.Connect(fmt.Sprintf("nats://%s:%s", os.Getenv("NATS_HOST"), os.Getenv("NATS_PORT")))
	if err != nil {
		log.Fatal("Failed to connect to NATS:", err)
	}

	log.Println("NATS connected successfully")
}

// PublishPostCreated publishes new post event
func PublishPostCreated(post models.Post) error {
	event := PostCreatedEvent{
		PostID:     post.ID,
		Content:    post.Content,
		AuthorID:   post.AuthorID,
		AuthorName: post.AuthorName,
		ImageURL:   post.ImageURL,
		Likes:      post.Likes,
		Timestamp:  post.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return NatsClient.Publish("post.created", eventJSON)
}

// PublishPostLiked publishes post liked event
func PublishPostLiked(post models.Post, userID string) error {
	event := PostLikedEvent{
		PostID:    post.ID,
		UserID:    userID,
		Likes:     post.Likes,
		Timestamp: post.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return NatsClient.Publish("post.liked", eventJSON)
}

// SubscribeToPosts subscribes to post events
func SubscribeToPosts(handler func([]byte)) (*nats.Subscription, error) {
	return NatsClient.Subscribe("post.*", func(msg *nats.Msg) {
		handler(msg.Data)
	})
}

// Event structures
type PostCreatedEvent struct {
	PostID     string  `json:"post_id"`
	Content    string  `json:"content"`
	AuthorID   string  `json:"author_id"`
	AuthorName string  `json:"author_name"`
	ImageURL   *string `json:"image_url"`
	Likes      int     `json:"likes"`
	Timestamp  string  `json:"timestamp"`
}

type PostLikedEvent struct {
	PostID    string `json:"post_id"`
	UserID    string `json:"user_id"`
	Likes     int    `json:"likes"`
	Timestamp string `json:"timestamp"`
}
