package appcatalog

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v2/resource"
	"github.com/giantswarm/operatorkit/v2/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v2/resource/wrapper/retryresource"

	"github.com/giantswarm/app-operator/v2/service/controller/appcatalog/resource/empty"
)

type appCatalogResourcesConfig struct {
	// Dependencies.
	Logger micrologger.Logger
}

// NewResourceSet returns a configured AppCatalog controller ResourceSet.
func newAppCatalogResources(config appCatalogResourcesConfig) ([]resource.Interface, error) {
	var err error

	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var emptyResource resource.Interface
	{
		emptyResource = empty.New()
	}

	resources := []resource.Interface{
		emptyResource,
	}

	{
		c := retryresource.WrapConfig{
			Logger: config.Logger,
		}

		resources, err = retryresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		c := metricsresource.WrapConfig{}
		resources, err = metricsresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resources, nil
}
