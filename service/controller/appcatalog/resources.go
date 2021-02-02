package appcatalog

import (
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/retryresource"

	"github.com/giantswarm/app-operator/v3/service/controller/appcatalog/resource/appcatalogentry"
	"github.com/giantswarm/app-operator/v3/service/controller/appcatalog/resource/appcatalogfinalizer"
)

type appCatalogResourcesConfig struct {
	// Dependencies.
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	// Settings.
	MaxEntriesPerApp int
	UniqueApp        bool
}

// NewResourceSet returns a configured AppCatalog controller ResourceSet.
func newAppCatalogResources(config appCatalogResourcesConfig) ([]resource.Interface, error) {
	var err error

	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var appCatalogFinalizerResource resource.Interface
	{
		c := appcatalogfinalizer.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}
		appCatalogFinalizerResource, err = appcatalogfinalizer.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
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
		appCatalogFinalizerResource,
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
