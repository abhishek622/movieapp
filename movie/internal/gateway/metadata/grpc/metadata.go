package grpc

import (
	"context"

	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/internal/grpcutil"
	"github.com/abhishek622/movieapp/metadata/pkg/model"
	"github.com/abhishek622/movieapp/pkg/discovery"
	"google.golang.org/grpc/credentials"
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
	resp, err := client.GetMetadata(ctx, &gen.GetMetadataRequest{MovieId: id})
	if err != nil {
		return nil, err
	}

	return model.MetadataFromProto(resp.Metadata), nil
}
