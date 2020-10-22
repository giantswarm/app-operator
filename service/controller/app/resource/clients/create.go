package clients

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/v2/service/controller/key"
)

// EnsureCreated adds g8s and k8s clients to the controller context based on the
// kubeconfig settings for the app CR.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
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
