package memory

import (
	"context"
	"sync"

	"github.com/abhishek622/movieapp/metadata/internal/repository"
	"github.com/abhishek622/movieapp/metadata/pkg/model"
	"go.opentelemetry.io/otel"
)

// Repository defines a memory movie matadata repository.
type Repository struct {
	sync.RWMutex
	data map[string]*model.Metadata
}

const tracerID = "metadata-repository-memory"

// New creates a new memory repository.
func New() *Repository {
	return &Repository{data: map[string]*model.Metadata{}}
}

// Get retrieves movie metadata for by movie id.
func (r *Repository) Get(ctx context.Context, id string) (*model.Metadata, error) {
	r.RLock()
	defer r.RUnlock()

	_, span := otel.Tracer(tracerID).Start(ctx, "Repository/Get")
	defer span.End()

	m, ok := r.data[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return m, nil
}

// Put adds movie metadata for a given movie id.
func (r *Repository) Put(ctx context.Context, id string, metadata *model.Metadata) error {
	r.Lock()
	defer r.Unlock()

	_, span := otel.Tracer(tracerID).Start(ctx, "Repository/Get")
	defer span.End()

	r.data[id] = metadata
	return nil
}
