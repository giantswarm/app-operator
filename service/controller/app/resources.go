package app

import (
	"time"

	"github.com/giantswarm/app/v4/pkg/values"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/crud"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/retryresource"
	"github.com/spf13/afero"

	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/appcatalog"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/appfinalizermigration"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/appnamespace"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/authtokenmigration"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/chart"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/chartcrd"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/chartoperator"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/clients"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/configmap"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/releasemigration"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/secret"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/status"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/tcnamespace"
	"github.com/giantswarm/app-operator/v4/service/controller/app/resource/validation"
	"github.com/giantswarm/app-operator/v4/service/internal/k8sclientcache"
)

type appResourcesConfig struct {
	// Dependencies.
	FileSystem     afero.Fs
	K8sClient      k8sclient.Interface
	K8sClientCache *k8sclientcache.Resource
	Logger         micrologger.Logger

	// Settings.
	ChartNamespace    string
	HTTPClientTimeout time.Duration
	ImageRegistry     string
	Provider          string
	UniqueApp         bool
}

func newAppResources(config appResourcesConfig) ([]resource.Interface, error) {
	var err error

	// Dependencies.
	if config.FileSystem == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Fs must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.K8sClientCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CachedK8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	// Settings.
	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}
	if config.HTTPClientTimeout == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.HTTPClientTimeout must not be empty", config)
	}
	if config.ImageRegistry == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ImageRegistry must not be empty", config)
	}
	if config.Provider == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}

	var valuesService *values.Values
	{
		c := values.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		valuesService, err = values.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appcatalogResource resource.Interface
	{
		c := appcatalog.Config{
			G8sClient: config.K8sClient.G8sClient(),
			Logger:    config.Logger,
		}
		appcatalogResource, err = appcatalog.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appFinalizerResource resource.Interface
	{
		c := appfinalizermigration.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}
		appFinalizerResource, err = appfinalizermigration.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appNamespaceResource resource.Interface
	{
		c := appnamespace.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}
		appNamespaceResource, err = appnamespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var authTokenMigrationResource resource.Interface
	{
		c := authtokenmigration.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}
		authTokenMigrationResource, err = authtokenmigration.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var chartOperatorResource resource.Interface
	{
		c := chartoperator.Config{
			FileSystem: config.FileSystem,
			G8sClient:  config.K8sClient.G8sClient(),
			K8sClient:  config.K8sClient.K8sClient(),
			Logger:     config.Logger,
			Values:     valuesService,
		}
		chartOperatorResource, err = chartoperator.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var chartResource resource.Interface
	{
		c := chart.Config{
			Logger: config.Logger,

			ChartNamespace: config.ChartNamespace,
		}

		ops, err := chart.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		chartResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var chartCRDResource resource.Interface
	{
		c := chartcrd.Config{
			Logger: config.Logger,
		}

		chartCRDResource, err = chartcrd.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var helmClient helmclient.Interface
	{
		c := helmclient.Config{
			Fs:         config.FileSystem,
			K8sClient:  config.K8sClient.K8sClient(),
			Logger:     config.Logger,
			RestClient: config.K8sClient.RESTClient(),
			RestConfig: config.K8sClient.RESTConfig(),

			HTTPClientTimeout: config.HTTPClientTimeout,
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clientsResource resource.Interface
	{
		c := clients.Config{
			HelmClient:     helmClient,
			K8sClient:      config.K8sClient,
			K8sClientCache: config.K8sClientCache,
			Logger:         config.Logger,
		}

		clientsResource, err = clients.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var configMapResource resource.Interface
	{
		c := configmap.Config{
			Logger: config.Logger,
			Values: valuesService,

			ChartNamespace: config.ChartNamespace,
		}

		ops, err := configmap.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		configMapResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var releaseMigrationResource resource.Interface
	{
		c := releasemigration.Config{
			Logger: config.Logger,

			ChartNamespace: config.ChartNamespace,
			ImageRegistry:  config.ImageRegistry,
		}

		releaseMigrationResource, err = releasemigration.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var secretResource resource.Interface
	{
		c := secret.Config{
			Logger: config.Logger,
			Values: valuesService,

			ChartNamespace: config.ChartNamespace,
		}

		ops, err := secret.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		secretResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var statusResource resource.Interface
	{
		c := status.Config{
			G8sClient: config.K8sClient.G8sClient(),
			Logger:    config.Logger,

			ChartNamespace: config.ChartNamespace,
		}

		statusResource, err = status.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var tcNamespaceResource resource.Interface
	{
		c := tcnamespace.Config{
			Logger: config.Logger,
		}

		tcNamespaceResource, err = tcnamespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var validationResource resource.Interface
	{
		c := validation.Config{
			G8sClient: config.K8sClient.G8sClient(),
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			Provider: config.Provider,
		}

		validationResource, err = validation.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		// validationResource checks CRs for validation errors and sets the CR status.
		validationResource,

		// appFinalizerResource check CRs for legacy finalizers and removes them.
		appFinalizerResource,

		// Following resources manage controller context information.
		appNamespaceResource,
		appcatalogResource,
		clientsResource,

		// authTokenMigrationResource deletes auth token secrets that are no
		// longer used.
		authTokenMigrationResource,

		// Following resources bootstrap chart-operator in workload clusters.
		tcNamespaceResource,
		chartCRDResource,
		chartOperatorResource,
		releaseMigrationResource,

		// Following resources process app CRs.
		configMapResource,
		secretResource,
		chartResource,
		statusResource,
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

func toCRUDResource(logger micrologger.Logger, ops crud.Interface) (*crud.Resource, error) {
	c := crud.ResourceConfig{
		Logger: logger,
		CRUD:   ops,
	}

	r, err := crud.NewResource(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r, nil
}
