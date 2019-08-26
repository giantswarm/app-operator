package chartoperator

import (
	"context"
	"fmt"

	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"

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

	// Check whether tenant cluster has a chart-operator helm release yet.
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding chart-operator release %#q in tenant cluster", release))

		_, err := cc.HelmClient.GetReleaseContent(ctx, release)
		if helmclient.IsReleaseNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find chart-perator release %#q in tenant cluster", release))
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing chart-operator release %#q in tenant cluster", release))

			err = r.installChartOperator(ctx, cr, cc.HelmClient)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed chart-operator release %#q in tenant cluster", release))
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found chart-operator release %#q", release))
		}
	}

	return nil
}
