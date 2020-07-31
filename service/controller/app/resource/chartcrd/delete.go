package chartcrd

import "context"

// EnsureDeleted is a no-op because the chart CRD in the tenant cluster is
// deleted with the tenant cluster resources.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}
