package chartoperator

import (
	"context"
	"fmt"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/key"
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

	// Resource is used to bootstrap chart-operator. So for other apps we can
	// skip this step.
	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to install chart-operator for %#q", key.AppName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is unavailable")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	// Check whether cluster has a chart-operator helm release yet.
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding release %#q", cr.Name))

		_, err := cc.Clients.Helm.GetReleaseContent(ctx, cr.Name)
		if helmclient.IsTillerNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "no healthy tiller pod found")

			// Tiller may not be healthy and we cannot continue without a connection
			// to Tiller. We will retry on next reconciliation loop.
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)

			return nil
		} else if helmclient.IsTillerOutdated(err) {
			// Tiller is upgraded by chart-operator. When we want to upgrade
			// Tiller we deploy a new version of chart-operator. So here we
			// can just cancel the resource.
			r.logger.LogCtx(ctx, "level", "debug", "message", "tiller pod is outdated")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		} else if tenant.IsAPINotAvailable(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available")

			// We should not hammer tenant API if it is not available, the tenant
			// cluster might be initializing. We will retry on next reconciliation
			// loop.
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)

			return nil
		} else if helmclient.IsReleaseNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find release %#q", cr.Name))
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing release %#q", cr.Name))

			err = r.installChartOperator(ctx, cr)
			if IsNotReady(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%#q not ready", cr.Name))

				// chart-operator installs the chart CRD in the cluster.
				// So if its not ready we cancel and retry on the next
				// reconciliation loop.
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
				reconciliationcanceledcontext.SetCanceled(ctx)

				return nil
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed release %#q", cr.Name))
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found release %#q", cr.Name))

			releaseContent, err := cc.Clients.Helm.GetReleaseContent(ctx, cr.Name)
			if err != nil {
				return microerror.Mask(err)
			}

			if releaseContent.Status == "FAILED" {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("release %#q failed to install", cr.Name))
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating release %#q", cr.Name))

				err = r.updateChartOperator(ctx, cr)
				if err != nil {
					return microerror.Mask(err)
				}

				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated release %#q", cr.Name))
			}
		}
	}

	return nil
}
