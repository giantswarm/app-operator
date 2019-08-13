package cordonchart

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	// Name is the identifier of the resource.
	Name = "cordonchartv1"
)

//Config represents the configuration used to create a new cordonchart resource.
type Config struct {
	// Dependencies.
	Logger micrologger.Logger

	// Settings.
	ChartNamespace string
}

// Resource implements the cordonchart resource.
type Resource struct {
	// Dependencies.
	logger micrologger.Logger

	// Settings.
	chartNamespace string
}

// New creates a new configured cordonchart resource.
func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}

	r := &Resource{
		logger:         config.Logger,
		chartNamespace: config.ChartNamespace,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}
