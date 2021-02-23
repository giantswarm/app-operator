package appnamespace

import (
	"context"

	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/microerror"
)

// EnsureDeleted checks whether the namespace this app CR belongs to
// is currently being deleted.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// If the app CR is in-cluster then we need to delete the resources even if
	// the namespace is being deleted.
	if key.InCluster(cr) {
		return nil
	}

	err = r.addNamespaceStatusToContext(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
