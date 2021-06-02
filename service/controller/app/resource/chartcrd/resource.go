package chartcrd

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app-operator/v4/service/internal/crdcache"
)

const (
	Name = "chartcrd"
)

type Config struct {
	CRDCache *crdcache.Resource
	Logger   micrologger.Logger
}

type Resource struct {
	crdCache *crdcache.Resource
	logger   micrologger.Logger
}

// New creates a new configured tcnamespace resource.
func New(config Config) (*Resource, error) {
	if config.CRDCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CRDCache must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		crdCache: config.CRDCache,
		logger:   config.Logger,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}
