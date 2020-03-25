package chartoperator

import (
	"context"
	"fmt"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
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
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to install chart-operator for %#q", key.AppName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if key.InCluster(cr) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q uses InCluster kubeconfig no need to install chart operator", key.AppName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling the resource")
		return nil
	}

	if cc.Status.TenantCluster.IsUnavailable {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is unavailable")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	// Check whether tenant cluster has a chart-operator helm release yet.
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding chart-operator release %#q in tenant cluster", release))

		_, err := cc.Clients.Helm.GetReleaseContent(ctx, key.Namespace(cr), release)
		if tenant.IsAPINotAvailable(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available")

			// We should not hammer tenant API if it is not available, the tenant
			// cluster might be initializing. We will retry on next reconciliation
			// loop.
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)

			return nil
		} else if helmclient.IsReleaseNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find chart-perator release %#q in tenant cluster", release))
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing chart-operator release %#q in tenant cluster", release))

			err = r.installChartOperator(ctx, cr)
			if IsNotReady(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "chart-operator not ready")

				// chart-operator installs the chart CRD in the tenant cluster.
				// So if its not ready we cancel and retry on the next
				// reconciliation loop.
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
				reconciliationcanceledcontext.SetCanceled(ctx)

				return nil
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed chart-operator release %#q in tenant cluster", release))
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found chart-operator release %#q", release))

			releaseContent, err := cc.Clients.Helm.GetReleaseContent(ctx, key.Namespace(cr), release)
			if err != nil {
				return microerror.Mask(err)
			}

			if releaseContent.Status == helmclient.StatusFailed {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("chart-operator release %#q failed to install", release))
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating a release %#q", release))

				err = r.updateChartOperator(ctx, cr)
				if err != nil {
					return microerror.Mask(err)
				}

				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated a release %#q", release))

			}
		}
	}

	return nil
}
