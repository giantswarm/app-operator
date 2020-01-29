package tiller

import (
	"context"

	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
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

func (r *Resource) ensureCRDCreated(ctx context.Context, k8sClient kubernetes.Interface) error {
	k8sClient.CRD


	return nil
}
