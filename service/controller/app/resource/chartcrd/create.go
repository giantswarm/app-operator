package chartcrd

import (
	"context"
	"time"

	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/app-operator/v6/pkg/crd"
	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
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

	// Resource is used to bootstrap chart-operator in workload clusters.
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
		r.logger.Debugf(ctx, "workload cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	r.logger.Debugf(ctx, "ensuring chart CRD in workload cluster %#q", key.ClusterID(cr))

	var chartCRD apiextensionsv1.CustomResourceDefinition

	err = yaml.Unmarshal([]byte(crd.ChartCRD()), &chartCRD)
	if err != nil {
		return microerror.Mask(err)
	}

	ch := make(chan error)

	go func() {
		err = cc.Clients.K8s.CRDClient().EnsureCreated(ctx, &chartCRD, backoff.NewMaxRetries(7, 1*time.Second))

		close(ch)
	}()

	select {
	case <-ch:
		// Fall through.
	case <-time.After(10 * time.Second):
		// Set status so we don't try to connect to the workload cluster
		// again in this reconciliation loop.
		cc.Status.ClusterStatus.IsUnavailable = true

		r.logger.Debugf(ctx, "timeout ensuring chart CRD")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if apierrors.IsAlreadyExists(err) {
		// fall through
	} else if tenant.IsAPINotAvailable(err) {
		// Set status so we don't try to connect to the workload cluster
		// again in this reconciliation loop.workload
		cc.Status.ClusterStatus.IsUnavailable = true

		r.logger.Debugf(ctx, "workload cluster not available")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensured chart CRD in workload cluster %#q", key.ClusterID(cr))

	return nil
}
