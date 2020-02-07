package chartcrd

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

const (
	Name = "chartcrdv1"
)

// Config represents the configuration used to create a new chardcrd resource.
type Config struct {
	// Dependencies.
	Logger micrologger.Logger
}

type Resource struct {
	// Dependencies.
	logger micrologger.Logger
}

// New creates a new configured tiller resource.
func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		logger: config.Logger,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

func (r *Resource) ensureChartCRDCreated(ctx context.Context) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring chardcrd exists in tenant cluster")

	err = cc.K8sClient.CRDClient().EnsureCreated(ctx, v1alpha1.NewChartCRD(), backoff.NewMaxRetries(7, 1*time.Second))
	if tenant.IsAPINotAvailable(err) {
		// We should not hammer tenant API if it is not available, the tenant cluster
		// might be initializing. We will retry on next reconciliation loop.
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is not available.")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured chardcrd exists in tenant cluster")

	return nil
}
