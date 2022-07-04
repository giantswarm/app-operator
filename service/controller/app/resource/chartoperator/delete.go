package chartoperator

import (
	"context"

	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/v7/service/controller/app/controllercontext"
)

func (r Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	// Resource is used to bootstrap chart-operator. So for other apps we can
	// skip this step.
	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.Debugf(ctx, "no need to install chart-operator for %#q", key.AppName(cr))
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if cc.Status.ClusterStatus.IsDeleting {
		r.logger.Debugf(ctx, "namespace %#q is being deleted, no need to reconcile resource", cr.Namespace)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "workload cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	err = r.uninstallChartOperator(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
