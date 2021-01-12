package chartcrd

import "context"

// EnsureDeleted is a no-op because the chart CRD in the workload cluster is
// deleted with the workload cluster resources.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}
