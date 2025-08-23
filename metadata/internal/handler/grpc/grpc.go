package grpc

import (
	"context"
	"errors"

	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/metadata/internal/controller/metadata"
	"github.com/abhishek622/movieapp/metadata/pkg/model"
	"github.com/uber-go/tally/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler defines a movie metadata gRPC handler.
type Handler struct {
	gen.UnimplementedMetadataServiceServer
	ctrl               *metadata.Controller
	getMetadataMetrics *EndpointMetrics
	putMetadataMetrics *EndpointMetrics
}

// New creates a new movie metadata gRPC handler.
func New(ctrl *metadata.Controller, scope tally.Scope) *Handler {
	return &Handler{ctrl: ctrl, getMetadataMetrics: newEndpointMetrics(scope, "GetMetadata"), putMetadataMetrics: newEndpointMetrics(scope, "PutMetadata")}
}

type EndpointMetrics struct {
	calls                 tally.Counter
	invalidArgumentErrors tally.Counter
	notFoundErrors        tally.Counter
	internalErrors        tally.Counter
	successes             tally.Counter
}

func newEndpointMetrics(scope tally.Scope, endpoint string) *EndpointMetrics {
	scope = scope.Tagged(map[string]string{"component": "handler", "endpoint": endpoint})
	return &EndpointMetrics{
		calls:                 scope.Counter("call"),
		invalidArgumentErrors: scope.Tagged(map[string]string{"error": "invalid_argument"}).Counter("error"),
		notFoundErrors:        scope.Tagged(map[string]string{"error": "not_found"}).Counter("error"),
		internalErrors:        scope.Tagged(map[string]string{"error": "internal"}).Counter("error"),
		successes:             scope.Counter("success"),
	}
}

// GetMetadata returns movie metadata.
func (h *Handler) GetMetadata(ctx context.Context, req *gen.GetMetadataRequest) (*gen.GetMetadataResponse, error) {
	h.getMetadataMetrics.calls.Inc(1)
	if req == nil || req.MovieId == "" {
		h.getMetadataMetrics.invalidArgumentErrors.Inc(1)
		return nil, status.Errorf(codes.InvalidArgument, "nil req or empty id")
	}
	m, err := h.ctrl.Get(ctx, req.MovieId)
	if err != nil && errors.Is(err, metadata.ErrNotFound) {
		h.getMetadataMetrics.notFoundErrors.Inc(1)
		return nil, status.Errorf(codes.NotFound, "%s", err.Error())
	} else if err != nil {
		h.getMetadataMetrics.internalErrors.Inc(1)
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}

	h.getMetadataMetrics.successes.Inc(1)
	return &gen.GetMetadataResponse{Metadata: model.MetadataToProto(m)}, nil
}

// PutMetadata puts movie metadata to repository.
func (h *Handler) PutMetadata(ctx context.Context, req *gen.PutMetadataRequest) (*gen.PutMetadataResponse, error) {
	h.getMetadataMetrics.calls.Inc(1)
	if req == nil || req.Metadata == nil {
		h.getMetadataMetrics.invalidArgumentErrors.Inc(1)
		return nil, status.Errorf(codes.InvalidArgument, "nil req or metadata")
	}
	if err := h.ctrl.Put(ctx, model.MetadataFromProto(req.Metadata)); err != nil {
		h.getMetadataMetrics.invalidArgumentErrors.Inc(1)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	h.getMetadataMetrics.successes.Inc(1)
	return &gen.PutMetadataResponse{}, nil
}
