package models

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID         string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Content    string         `json:"content" gorm:"not null"`
	AuthorID   string         `json:"author_id" gorm:"not null"`
	AuthorName string         `json:"author_name" gorm:"not null"`
	ImageURL   *string        `json:"image_url"`
	Likes      int            `json:"likes" gorm:"default:0"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type User struct {
	ID     string  `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name   string  `json:"name" gorm:"not null"`
	Avatar *string `json:"avatar"`
}

type PostLike struct {
	ID     string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PostID string `json:"post_id" gorm:"not null"`
	UserID string `json:"user_id" gorm:"not null"`
}
