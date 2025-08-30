package graphql

import (
	"context"
	"fmt"
	"muze/internal/auth"
	"muze/internal/cache"
	pb "muze/proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func NewResolver() *Resolver {
	// Connect to gRPC service
	conn, err := grpc.Dial("localhost:7001", grpc.WithInsecure())
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to gRPC service: %v", err))
	}

	client := pb.NewPostServiceClient(conn)
	return &Resolver{
		grpcClient: client,
	}
}

// Query resolvers
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) GetPosts(ctx context.Context, limit *int, offset *int) (*PostsResponse, error) {
	// Try cache first
	cachedPosts, err := cache.GetRecentPosts()
	if err == nil && len(cachedPosts) > 0 {
		var posts []*Post
		for _, p := range cachedPosts {
			posts = append(posts, &Post{
				ID:        p.ID,
				Content:   p.Content,
				Author:    &User{ID: p.AuthorID, Name: p.AuthorName},
				Likes:     p.Likes,
				Timestamp: p.CreatedAt.Format(time.RFC3339),
				ImageURL:  p.ImageURL,
			})
		}
		return &PostsResponse{Posts: posts, Total: len(posts)}, nil
	}

	// If not in cache, get from gRPC service
	req := &pb.GetPostsRequest{}
	if limit != nil {
		req.Limit = int32(*limit)
	}
	if offset != nil {
		req.Offset = int32(*offset)
	}

	resp, err := r.grpcClient.GetPosts(ctx, req)
	if err != nil {
		return nil, err
	}

	var posts []*Post
	for _, p := range resp.Posts {
		var imageURL *string
		if p.ImageUrl != nil {
			imageURL = &p.ImageUrl.Value
		}
		posts = append(posts, &Post{
			ID:        p.Id,
			Content:   p.Content,
			Author:    &User{ID: p.AuthorId, Name: p.AuthorName},
			Likes:     int(p.Likes),
			Timestamp: p.Timestamp,
			ImageURL:  imageURL,
		})
	}

	return &PostsResponse{Posts: posts, Total: int(resp.Total)}, nil
}

func (r *queryResolver) GetPostById(ctx context.Context, id string) (*Post, error) {
	// Try cache first
	cachedPost, err := cache.GetCachedPost(id)
	if err == nil {
		return &Post{
			ID:        cachedPost.ID,
			Content:   cachedPost.Content,
			Author:    &User{ID: cachedPost.AuthorID, Name: cachedPost.AuthorName},
			Likes:     cachedPost.Likes,
			Timestamp: cachedPost.CreatedAt.Format(time.RFC3339),
			ImageURL:  cachedPost.ImageURL,
		}, nil
	}

	// If not in cache, get from gRPC service
	resp, err := r.grpcClient.GetPostById(ctx, &pb.GetPostByIdRequest{Id: id})
	if err != nil {
		return nil, err
	}

	var imageURL *string
	if resp.ImageUrl != nil {
		imageURL = &resp.ImageUrl.Value
	}
	return &Post{
		ID:        resp.Id,
		Content:   resp.Content,
		Author:    &User{ID: resp.AuthorId, Name: resp.AuthorName},
		Likes:     int(resp.Likes),
		Timestamp: resp.Timestamp,
		ImageURL:  imageURL,
	}, nil
}

// Mutation resolvers
func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreatePost(ctx context.Context, content string, imageURL *string) (*Post, error) {
	// Get user from context (JWT token)
	claims, err := auth.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %v", err)
	}

	// Convert ImageURL to protobuf format
	var protoImageURL *wrapperspb.StringValue
	if imageURL != nil {
		protoImageURL = wrapperspb.String(*imageURL)
	}

	// Call gRPC service
	resp, err := r.grpcClient.CreatePost(ctx, &pb.CreatePostRequest{
		Content:  content,
		AuthorId: claims.UserID,
		ImageUrl: protoImageURL,
	})
	if err != nil {
		return nil, err
	}

	var respImageURL *string
	if resp.ImageUrl != nil {
		respImageURL = &resp.ImageUrl.Value
	}
	return &Post{
		ID:        resp.Id,
		Content:   resp.Content,
		Author:    &User{ID: resp.AuthorId, Name: resp.AuthorName},
		Likes:     int(resp.Likes),
		Timestamp: resp.Timestamp,
		ImageURL:  respImageURL,
	}, nil
}

func (r *mutationResolver) LikePost(ctx context.Context, postID string) (*Post, error) {
	// Get user from context (JWT token)
	claims, err := auth.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %v", err)
	}

	// Call gRPC service
	resp, err := r.grpcClient.LikePost(ctx, &pb.LikePostRequest{
		PostId: postID,
		UserId: claims.UserID,
	})
	if err != nil {
		return nil, err
	}

	var respImageURL *string
	if resp.ImageUrl != nil {
		respImageURL = &resp.ImageUrl.Value
	}
	return &Post{
		ID:        resp.Id,
		Content:   resp.Content,
		Author:    &User{ID: resp.AuthorId, Name: resp.AuthorName},
		Likes:     int(resp.Likes),
		Timestamp: resp.Timestamp,
		ImageURL:  respImageURL,
	}, nil
}

// GraphQL types
type Post struct {
	ID        string  `json:"id"`
	Content   string  `json:"content"`
	Author    *User   `json:"author"`
	Likes     int     `json:"likes"`
	Timestamp string  `json:"timestamp"`
	ImageURL  *string `json:"imageUrl"`
}

type User struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Avatar *string `json:"avatar"`
}

type PostsResponse struct {
	Posts []*Post `json:"posts"`
	Total int     `json:"total"`
}

type QueryResolver interface {
	GetPosts(ctx context.Context, limit *int, offset *int) (*PostsResponse, error)
	GetPostById(ctx context.Context, id string) (*Post, error)
}

type MutationResolver interface {
	CreatePost(ctx context.Context, content string, imageURL *string) (*Post, error)
	LikePost(ctx context.Context, postID string) (*Post, error)
}
