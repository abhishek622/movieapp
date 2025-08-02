package testutil

import (
	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/movie/internal/controller/movie"
	metadatagateway "github.com/abhishek622/movieapp/movie/internal/gateway/metadata/grpc"
	ratinggateway "github.com/abhishek622/movieapp/movie/internal/gateway/rating/grpc"
	grpchandler "github.com/abhishek622/movieapp/movie/internal/handler/grpc"
	"github.com/abhishek622/movieapp/pkg/discovery"
)

// NewTestMovieGRPCServer creates a new movie gRPC server to be used in tests.
func NewTestMovieGRPCServer(registry discovery.Registry) gen.MovieServiceServer {
	metadataGateway := metadatagateway.New(registry)
	ratingGateway := ratinggateway.New(registry)
	ctrl := movie.New(ratingGateway, metadataGateway)
	return grpchandler.New(ctrl)
}
