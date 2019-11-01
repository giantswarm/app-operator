package namespace

import (
	"context"

	"github.com/giantswarm/operatorkit/resource/crud"
)

// ApplyDeleteChange is a no-op because the namespace in the tenant cluster is
// deleted with the tenant cluster resources.
func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	return nil
}

// NewDeletePatch is a no-op because the namespace in the tenant cluster is
// deleted with the tenant cluster resources.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	return nil, nil
}
