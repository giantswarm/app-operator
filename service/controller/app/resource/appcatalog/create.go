package appcatalog

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/v2/service/controller/key"
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
