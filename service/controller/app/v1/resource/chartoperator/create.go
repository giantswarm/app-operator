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

	// Check whether TC has chart-operator deployment yet.
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding chartoperator release %#q in tenant cluster", chartOperatorRelease))
		_, err := cc.HelmClient.GetReleaseContent(ctx, chartOperatorRelease)
		if helmclient.IsReleaseNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no chartoperator release %#q in tenant cluster", chartOperatorRelease))

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installaing chartoperator release %#q in tenant cluster", chartOperatorRelease))
			err = r.installChartOperator(ctx, cr, cc.HelmClient)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed chartoperator release %#q in tenant cluster", chartOperatorRelease))
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found chartoperator release %#q", chartOperatorRelease))
		}
	}
	return nil
}
