package testutil

import (
	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/metadata/internal/controller/metadata"
	grpchandler "github.com/abhishek622/movieapp/metadata/internal/handler/grpc"
	"github.com/abhishek622/movieapp/metadata/internal/repository/memory"
)

// NewTestMetadataGRPCServer creates a new metadata gRPC server to be used in tests.
func NewTestMetadataGRPCServer() gen.MetadataServiceServer {
	r := memory.New()
	ctrl := metadata.New(r)
	return grpchandler.New(ctrl)
}
