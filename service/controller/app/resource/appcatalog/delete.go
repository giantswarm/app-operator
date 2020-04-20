package appcatalog

import (
	"context"
)

// EnsureDeleted is a no-op because the app CR in the control plane would be deleted.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}
