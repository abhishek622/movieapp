package grpc

import (
	"context"

	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/internal/grpcutil"
	"github.com/abhishek622/movieapp/metadata/pkg/model"
	"github.com/abhishek622/movieapp/pkg/discovery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type Gateway struct {
	registry discovery.Registry
	creds    credentials.TransportCredentials
}

func New(registry discovery.Registry, creds credentials.TransportCredentials) *Gateway {
	return &Gateway{registry, creds}
}

func (g *Gateway) Get(ctx context.Context, id string) (*model.Metadata, error) {
	conn, err := grpcutil.ServiceConnection(ctx, "metadata", g.registry, g.creds)
	if err != nil {
		return nil, err
	}

	defer conn.Close()
	client := gen.NewMetadataServiceClient(conn)
	const maxRetries = 5
	for range maxRetries {
		var resp *gen.GetMetadataResponse
		resp, err = client.GetMetadata(ctx, &gen.GetMetadataRequest{MovieId: id})
		if err != nil {
			if shouldRetry(err) {
				continue
			}
			return nil, err
		}

		return model.MetadataFromProto(resp.Metadata), nil
	}

	return nil, err
}

func shouldRetry(err error) bool {
	e, ok := status.FromError(err)
	if !ok {
		return false
	}

	return e.Code() == codes.DeadlineExceeded || e.Code() == codes.ResourceExhausted || e.Code() == codes.Unavailable
}
