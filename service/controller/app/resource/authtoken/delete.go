package authtoken

import "context"

func (r Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	// no-op
	return nil
}
