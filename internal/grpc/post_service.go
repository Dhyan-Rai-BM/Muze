package grpc

import (
	"context"
	"log"
	"muze/internal/cache"
	"muze/internal/database"
	"muze/internal/messaging"
	"muze/internal/models"
	pb "muze/proto"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gorm.io/gorm"
)

type PostServer struct {
	pb.UnimplementedPostServiceServer
	db *gorm.DB
}

func NewPostServer() *PostServer {
	return &PostServer{
		db: database.DB,
	}
}

// Helper function to convert Go model to protobuf Post
func convertToProtoPost(post models.Post) *pb.Post {
	var imageURL *wrapperspb.StringValue
	if post.ImageURL != nil {
		imageURL = wrapperspb.String(*post.ImageURL)
	}

	return &pb.Post{
		Id:         post.ID,
		Content:    post.Content,
		AuthorId:   post.AuthorID,
		AuthorName: post.AuthorName,
		ImageUrl:   imageURL,
		Likes:      int32(post.Likes),
		Timestamp:  post.CreatedAt.Format(time.RFC3339),
	}
}

func (s *PostServer) GetPosts(ctx context.Context, req *pb.GetPostsRequest) (*pb.GetPostsResponse, error) {
	// Try cache first
	cachedPosts, err := cache.GetRecentPosts()
	if err == nil && len(cachedPosts) > 0 {
		// Convert to protobuf format
		var pbPosts []*pb.Post
		for _, post := range cachedPosts {
			pbPosts = append(pbPosts, convertToProtoPost(post))
		}
		return &pb.GetPostsResponse{Posts: pbPosts, Total: int32(len(pbPosts))}, nil
	}

	// If not in cache, get from database
	var posts []models.Post
	var total int64

	query := s.db.Model(&models.Post{})
	query.Count(&total)

	if req.Limit > 0 {
		query = query.Limit(int(req.Limit))
	}
	if req.Offset > 0 {
		query = query.Offset(int(req.Offset))
	}

	err = query.Order("created_at DESC").Find(&posts).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get posts: %v", err)
	}

	// Cache the results (exactly 10 most recent posts)
	if len(posts) > 0 {
		cachePosts := posts
		if len(posts) > 10 {
			cachePosts = posts[:10]
		}
		cache.CacheRecentPosts(cachePosts)
	}

	// Convert to protobuf format
	var pbPosts []*pb.Post
	for _, post := range posts {
		pbPosts = append(pbPosts, convertToProtoPost(post))
	}

	return &pb.GetPostsResponse{Posts: pbPosts, Total: int32(total)}, nil
}

func (s *PostServer) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.Post, error) {
	// Validate input
	if req.Content == "" {
		return nil, status.Errorf(codes.InvalidArgument, "content cannot be empty")
	}

	// Convert ImageUrl from protobuf to Go model
	var modelImageURL *string
	if req.ImageUrl != nil {
		modelImageURL = &req.ImageUrl.Value
	}

	// Create post
	post := models.Post{
		Content:    req.Content,
		AuthorID:   req.AuthorId,
		AuthorName: "User " + req.AuthorId, // In real app, get from user service
		ImageURL:   modelImageURL,
		Likes:      0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := s.db.Create(&post).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create post: %v", err)
	}

	// Cache the post
	cache.CachePost(post)

	// Publish to NATS for real-time updates
	messaging.PublishPostCreated(post)

	// Return response
	return convertToProtoPost(post), nil
}

func (s *PostServer) GetPostById(ctx context.Context, req *pb.GetPostByIdRequest) (*pb.Post, error) {
	// Try cache first
	cachedPost, err := cache.GetCachedPost(req.Id)
	if err == nil {
		return convertToProtoPost(*cachedPost), nil
	}

	// If not in cache, get from database
	var post models.Post
	err = s.db.Where("id = ?", req.Id).First(&post).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "post not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get post: %v", err)
	}

	// Cache the post
	cache.CachePost(post)

	return convertToProtoPost(post), nil
}

func (s *PostServer) LikePost(ctx context.Context, req *pb.LikePostRequest) (*pb.Post, error) {
	// Check if user already liked the post
	var existingLike models.PostLike
	err := s.db.Where("post_id = ? AND user_id = ?", req.PostId, req.UserId).First(&existingLike).Error
	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "user already liked this post")
	}

	// Start transaction
	tx := s.db.Begin()

	// Create like record
	like := models.PostLike{
		PostID: req.PostId,
		UserID: req.UserId,
	}
	err = tx.Create(&like).Error
	if err != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "failed to create like: %v", err)
	}

	// Update post likes count
	var post models.Post
	err = tx.Where("id = ?", req.PostId).First(&post).Error
	if err != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.NotFound, "post not found")
	}

	post.Likes++
	post.UpdatedAt = time.Now()
	err = tx.Save(&post).Error
	if err != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "failed to update post: %v", err)
	}

	// Commit transaction
	err = tx.Commit().Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	// Invalidate cache
	cache.InvalidatePostCache(req.PostId)

	// Publish to NATS for real-time updates
	messaging.PublishPostLiked(post, req.UserId)

	return convertToProtoPost(post), nil
}

func (s *PostServer) StreamPosts(req *pb.StreamPostsRequest, stream pb.PostService_StreamPostsServer) error {
	// Subscribe to NATS events
	subscription, err := messaging.SubscribeToPosts(func(data []byte) {
		// Parse the event and send to stream
		// This is a simplified version - in real app, you'd parse the event properly
		post := &pb.Post{
			Id:        "streamed-post",
			Content:   "Real-time post update",
			Timestamp: time.Now().Format(time.RFC3339),
		}

		if err := stream.Send(post); err != nil {
			log.Printf("Failed to send stream post: %v", err)
		}
	})

	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe to posts: %v", err)
	}
	defer subscription.Unsubscribe()

	// Keep the stream alive
	<-stream.Context().Done()
	return nil
}
