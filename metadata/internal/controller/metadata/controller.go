package metadata

import (
	"context"
	"errors"
	"fmt"

	"github.com/abhishek622/movieapp/metadata/internal/repository"
	"github.com/abhishek622/movieapp/metadata/pkg/model"
)

var ErrNotFound = errors.New("not found")

type metadataRepository interface {
	Get(ctx context.Context, id string) (*model.Metadata, error)
	Put(ctx context.Context, id string, metadata *model.Metadata) error
}

type Controller struct {
	repo  metadataRepository
	cache metadataRepository
}

func New(repo metadataRepository, cache metadataRepository) *Controller {
	return &Controller{repo, cache}
}

func (c *Controller) Get(ctx context.Context, id string) (*model.Metadata, error) {
	cacheRes, err := c.cache.Get(ctx, id)
	if err != nil {
		fmt.Println("Returning metadata from a cache for " + id)
		return cacheRes, nil
	}
	res, err := c.repo.Get(ctx, id)
	if err != nil && errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err := c.cache.Put(ctx, id, res); err != nil {
		fmt.Println("Error updating cache: " + err.Error())
	}

	return res, err

}
