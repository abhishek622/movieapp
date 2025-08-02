package testutil

import (
	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/rating/internal/controller/rating"
	grpchandler "github.com/abhishek622/movieapp/rating/internal/handler/grpc"
	"github.com/abhishek622/movieapp/rating/internal/repository/memory"
)

func NewTestRatingGRPCServer() gen.RatingServiceServer {
	r := memory.New()
	ctrl := rating.New(r, nil)
	return grpchandler.New(ctrl)
}
