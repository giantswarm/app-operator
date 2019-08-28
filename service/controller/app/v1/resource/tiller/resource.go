package tiller

import (
	"context"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"github.com/giantswarm/tenantcluster"
)

const (
	Name = "tillerv1"
)

// Config represents the configuration used to create a new tiller resource.
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

func (r *Resource) ensureTillerInstalled(ctx context.Context, helmClient helmclient.Interface) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring tiller is installed")

	values := []string{
		"spec.template.spec.priorityClassName=giantswarm-critical",
		"spec.template.spec.tolerations[0].effect=NoSchedule",
		"spec.template.spec.tolerations[0].key=node-role.kubernetes.io/master",
		"spec.template.spec.tolerations[0].operator=Exists",
	}

	var err error

	err = helmClient.EnsureTillerInstalledWithValues(ctx, values)
	if tenantcluster.IsTimeout(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "timeout fetching certificates")

		// A timeout error here means that the app-operator certificate
		// for the current tenant cluster was not found. We can't continue
		// without a Helm client. We will retry during the next execution, when
		// the certificate might be available.
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)

		return nil
	} else if helmclient.IsTillerNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "no healthy tiller pod found")

		// Tiller may not be healthy and we cannot continue without a connection
		// to Tiller. We will retry on next reconciliation loop.
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)

		return nil
	} else if tenant.IsAPINotAvailable(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available")

		// We should not hammer tenant API if it is not available, the tenant
		// cluster might be initializing. We will retry on next reconciliation
		// loop.
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured tiller is installed")
	return nil
}
