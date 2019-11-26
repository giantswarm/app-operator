package tiller

import (
	"context"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
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
		"spec.template.spec.tolerations[0].key=node.kubernetes.io/master",
		"spec.template.spec.tolerations[0].operator=Exists",
	}

	var err error

	err = helmClient.EnsureTillerInstalledWithValues(ctx, values)
	if helmclient.IsTillerNotFound(err) {
		// Tiller may not be healthy and we cannot continue without a connection
		// to Tiller. We will retry on next reconciliation loop.
		r.logger.LogCtx(ctx, "level", "debug", "message", "no healthy tiller pod found")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	} else if helmclient.IsTillerNotRunningError(err) {
		// Can't find a tiller pod in starting phase. We will retry on next reconciliation loop.
		r.logger.LogCtx(ctx, "level", "debug", "message", "no running tiller pod")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	} else if helmclient.IsTooManyResults(err) {
		// Too many tiller pods due to upgrade. We will retry on next reconciliation loop.
		r.logger.LogCtx(ctx, "level", "debug", "message", "currently too many tiller pods due to upgrade")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	} else if tenant.IsAPINotAvailable(err) {
		// We should not hammer tenant API if it is not available, the tenant
		// cluster might be initializing. We will retry on next reconciliation
		// loop.
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured tiller is installed")
	return nil
}
