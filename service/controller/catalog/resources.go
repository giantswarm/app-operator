package catalog

import (
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/retryresource"

	"github.com/giantswarm/app-operator/v4/service/controller/catalog/resource/appcatalogentry"
)

type catalogResourcesConfig struct {
	// Dependencies.
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	// Settings.
	MaxEntriesPerApp int
	UniqueApp        bool
}

// newCatalogResources returns a configured Catalog controller ResourceSet.
func newCatalogResources(config catalogResourcesConfig) ([]resource.Interface, error) {
	var err error

	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var appCatalogEntryResource resource.Interface
	{
		c := appcatalogentry.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			MaxEntriesPerApp: config.MaxEntriesPerApp,
			UniqueApp:        config.UniqueApp,
		}

		appCatalogEntryResource, err = appcatalogentry.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		appCatalogEntryResource,
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