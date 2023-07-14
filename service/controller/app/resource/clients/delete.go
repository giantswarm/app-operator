package clients

import (
	"context"

	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/microerror"
)

// EnsureDeleted adds g8s and k8s clients to the controller context based on the
// kubeconfig settings for the app CR.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.addClientsToContext(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
