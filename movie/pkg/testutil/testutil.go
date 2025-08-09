package testutil

import (
	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/movie/internal/controller/movie"
	metadatagateway "github.com/abhishek622/movieapp/movie/internal/gateway/metadata/grpc"
	ratinggateway "github.com/abhishek622/movieapp/movie/internal/gateway/rating/grpc"
	grpchandler "github.com/abhishek622/movieapp/movie/internal/handler/grpc"
	"github.com/abhishek622/movieapp/pkg/discovery"
	"google.golang.org/grpc/credentials/insecure"
)

// NewTestMovieGRPCServer creates a new movie gRPC server to be used in tests.
func NewTestMovieGRPCServer(registry discovery.Registry) gen.MovieServiceServer {
	metadataGateway := metadatagateway.New(registry, insecure.NewCredentials())
	ratingGateway := ratinggateway.New(registry, insecure.NewCredentials())
	ctrl := movie.New(ratingGateway, metadataGateway)
	return grpchandler.New(ctrl)
}
