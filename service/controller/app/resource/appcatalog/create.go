package appcatalog

import (
	"context"

	"github.com/giantswarm/app/v3/pkg/key"
	"github.com/giantswarm/microerror"
)

// EnsureCreated gets the appCatalog CR specified in the provided app CR.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.getCatalogForApp(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
