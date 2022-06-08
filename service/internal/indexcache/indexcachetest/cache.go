package indexcachetest

import (
	"context"

	"github.com/giantswarm/app-operator/v6/service/internal/indexcache"
)

type Config struct {
	GetIndexError    error
	GetIndexResponse *indexcache.Index
}

type Resource struct {
	getIndexError    error
	getIndexResponse *indexcache.Index
}

func New(config Config) *Resource {
	r := &Resource{
		getIndexError:    config.GetIndexError,
		getIndexResponse: config.GetIndexResponse,
	}

	return r
}

func (r *Resource) GetIndex(ctx context.Context, url string) (*indexcache.Index, error) {
	if r.getIndexError != nil {
		return nil, r.getIndexError
	}
	if r.getIndexResponse != nil {
		return r.getIndexResponse, nil
	}

	return nil, nil
}
