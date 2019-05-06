package empty

import "context"

func New() *Resource {
	return &Resource{}
}

type Resource struct{}

// EnsureCreated is not implemented for the empty resource.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	return nil
}

// EnsureDeleted is not implemented for the empty resource.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

func (r *Resource) Name() string {
	return "empty"
}
