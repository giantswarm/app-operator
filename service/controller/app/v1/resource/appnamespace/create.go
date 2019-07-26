package appnamespace

import "context"

// EnsureCreated is not implemented because we only need to consider
// the status of the namespace on deletion events.
func (*Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	return nil
}
