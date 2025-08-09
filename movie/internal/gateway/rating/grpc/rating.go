package grpc

import (
	"context"

	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/internal/grpcutil"
	"github.com/abhishek622/movieapp/pkg/discovery"
	"github.com/abhishek622/movieapp/rating/pkg/model"
	"google.golang.org/grpc/credentials"
)

// Gateway defines an gRPC gateway for a rating service.
type Gateway struct {
	registry discovery.Registry
	creds    credentials.TransportCredentials
}

// New creates a new gRPC gateway for a rating service.
func New(registry discovery.Registry, creds credentials.TransportCredentials) *Gateway {
	return &Gateway{registry, creds}
}

// GetAggregatedRating returns the aggregated rating for a record or ErrNotFound if there are no ratings for it.
func (g *Gateway) GetAggregatedRating(ctx context.Context, recordID model.RecordID, recordType model.RecordType) (float64, error) {
	conn, err := grpcutil.ServiceConnection(ctx, "rating", g.registry, g.creds)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	client := gen.NewRatingServiceClient(conn)
	resp, err := client.GetAggregatedRating(ctx, &gen.GetAggregatedRatingRequest{RecordId: string(recordID), RecordType: string(recordType)})
	if err != nil {
		return 0, err
	}
	return resp.RatingValue, nil
}
