package graphql

import (
	pb "muze/proto"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	grpcClient pb.PostServiceClient
}
