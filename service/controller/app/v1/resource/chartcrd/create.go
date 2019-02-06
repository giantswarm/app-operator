package chartcrd

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error
	ctlCtx, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring chart crd creation on tenant cluster")

	crdBackoff := backoff.NewMaxRetries(3, 1*time.Second)
	err = ctlCtx.CRDClient.EnsureCreated(ctx, v1alpha1.NewChartCRD(), crdBackoff)
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "error", "failed to ensured chart crd creation on tenant cluster")
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured chart crd creation on tenant cluster")

	return nil
}
