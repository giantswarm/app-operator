package pause

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/app/v3/pkg/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/reconciliationcanceledcontext"
)

const (
	Name = "pause"
)

// Config represents the configuration used to create a new chartstatus resource.
type Config struct {
	Logger micrologger.Logger
}

// Resource implements the chartstatus resource.
type Resource struct {
	logger micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		logger: config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) ensure(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(cr.GetAnnotations()) == 0 {
		return nil
	}

	v := cr.GetAnnotations()[annotation.AppOperatorPaused]
	if v == "true" {
		r.logger.Debugf(ctx, "canceling reconciliation due to %#q annotation set to %#q", annotation.AppOperatorPaused, v)
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}

	return nil
}
