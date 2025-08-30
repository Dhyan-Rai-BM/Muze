package main

import (
	"log"
	"muze/internal/cache"
	"muze/internal/database"
	grpcService "muze/internal/grpc"
	"muze/internal/messaging"
	pb "muze/proto"
	"net"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize services
	database.InitDB()
	cache.InitRedis()
	messaging.InitNATS()

	// Create gRPC server
	server := grpc.NewServer()

	// Register Post service
	postServer := grpcService.NewPostServer()
	pb.RegisterPostServiceServer(server, postServer)

	// Enable reflection for debugging
	reflection.Register(server)

	// Start server
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "7001"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("gRPC server starting on port %s", port)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
