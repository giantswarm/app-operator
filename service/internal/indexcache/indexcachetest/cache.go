package indexcachetest

import (
	"context"
	"fmt"

	"github.com/giantswarm/app-operator/v7/service/internal/indexcache"
)

type Config struct {
	GetIndexError    error
	GetIndexResponse *indexcache.Index
}

type Resource struct {
	getIndexError    error
	getIndexResponse *indexcache.Index
}

type MapResource struct {
	indices map[string]*Resource
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

func NewMap(config map[string]Config) *MapResource {
	m := map[string]*Resource{}
	for url, cfg := range config {
		m[url] = New(cfg)
	}
	r := &MapResource{
		indices: m,
	}
	return r
}

func (r *MapResource) GetIndex(ctx context.Context, url string) (*indexcache.Index, error) {
	idx, ok := r.indices[url]
	if !ok {
		return nil, fmt.Errorf("index %s not found", url)
	}
	if idx.getIndexError != nil {
		return nil, idx.getIndexError
	}
	if idx.getIndexResponse != nil {
		return idx.getIndexResponse, nil
	}
	return nil, nil
}
