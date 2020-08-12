package appnamespace

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/v2/service/controller/app/key"
)

// EnsureDeleted checks whether the namespace this app CR belongs to
// is currently being deleted.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.addNamespaceStatusToContext(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
