package namespace

import (
	"context"

	"github.com/giantswarm/operatorkit/controller"
)

// ApplyDeleteChange is a no-op because the namespace in the tenant cluster is
// deleted with the tenant cluster resources.
func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	return nil
}

// NewDeletePatch is a no-op because the namespace in the tenant cluster is
// deleted with the tenant cluster resources.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	return nil, nil
}
