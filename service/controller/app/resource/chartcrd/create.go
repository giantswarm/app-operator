package chartcrd

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/crd"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/app-operator/v3/service/controller/app/controllercontext"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	// Resource is used to bootstrap chart-operator in tenant clusters.
	// So for other apps we can skip this step.
	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.Debugf(ctx, "no need to create namespace for %#q", key.AppName(cr))
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if key.InCluster(cr) {
		r.logger.Debugf(ctx, "app %#q in %#q uses InCluster kubeconfig no need to ensure chart CRD", cr.Name, cr.Namespace)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "tenant cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	r.logger.Debugf(ctx, "ensuring chart CRD in tenant cluster %#q", key.ClusterID(cr))

	ch := make(chan error)

	go func() {
		err = cc.Clients.K8s.CRDClient().EnsureCreated(ctx, crd.LoadV1("application.giantswarm.io", "Chart"), backoff.NewMaxRetries(7, 1*time.Second))

		close(ch)
	}()

	select {
	case <-ch:
		// Fall through.
	case <-time.After(10 * time.Second):
		// Set status so we don't try to connect to the tenant cluster
		// again in this reconciliation loop.
		cc.Status.ClusterStatus.IsUnavailable = true

		r.logger.Debugf(ctx, "timeout ensuring chart CRD")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if apierrors.IsAlreadyExists(err) {
		// fall through
	} else if tenant.IsAPINotAvailable(err) {
		// Set status so we don't try to connect to the tenant cluster
		// again in this reconciliation loop.
		cc.Status.ClusterStatus.IsUnavailable = true

		r.logger.Debugf(ctx, "tenant cluster not available")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensured chart CRD in tenant cluster %#q", key.ClusterID(cr))

	return nil
}
