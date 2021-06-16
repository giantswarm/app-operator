package catalog

import (
	"context"
)

// EnsureDeleted is a no-op because the app CR in the management cluster would be deleted.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}
