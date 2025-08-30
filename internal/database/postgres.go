package database

import (
	"fmt"
	"log"
	"muze/internal/models"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate tables
	err = DB.AutoMigrate(&models.Post{}, &models.User{}, &models.PostLike{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Create indexes for performance optimization
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_posts_author_id ON posts(author_id)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_post_likes_post_id ON post_likes(post_id)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_post_likes_user_id ON post_likes(user_id)")

	log.Println("Database connected and migrated successfully")
}
