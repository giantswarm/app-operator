package app

import (
	"context"

	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/crud"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"github.com/spf13/afero"

	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/resource/appcatalog"
	"github.com/giantswarm/app-operator/service/controller/app/resource/appnamespace"
	"github.com/giantswarm/app-operator/service/controller/app/resource/chart"
	"github.com/giantswarm/app-operator/service/controller/app/resource/chartoperator"
	"github.com/giantswarm/app-operator/service/controller/app/resource/clients"
	"github.com/giantswarm/app-operator/service/controller/app/resource/configmap"
	"github.com/giantswarm/app-operator/service/controller/app/resource/secret"
	"github.com/giantswarm/app-operator/service/controller/app/resource/status"
	"github.com/giantswarm/app-operator/service/controller/app/resource/tcnamespace"
	"github.com/giantswarm/app-operator/service/controller/app/resource/tiller"
	"github.com/giantswarm/app-operator/service/controller/app/resource/validation"
	"github.com/giantswarm/app-operator/service/controller/app/values"
)

// ResourceSetConfig contains necessary dependencies and settings for
// AppConfig controller ResourceSet configuration.
type ResourceSetConfig struct {
	// Dependencies.
	FileSystem afero.Fs
	K8sClient  k8sclient.Interface
	Logger     micrologger.Logger

	// Settings.
	ChartNamespace string
	ImageRegistry  string
	UniqueApp      bool
}

// NewResourceSet returns a configured App controller ResourceSet.
func NewResourceSet(config ResourceSetConfig) (*controller.ResourceSet, error) {
	var err error

	// Dependencies.
	if config.FileSystem == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Fs must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	// Settings.
	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}
	if config.ImageRegistry == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ImageRegistry must not be empty", config)
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

	var clientsResource resource.Interface
	{
		c := clients.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			ImageRegistry: config.ImageRegistry,
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

	var tillerResource resource.Interface
	{
		c := tiller.Config{
			Logger: config.Logger,
		}
		tillerResource, err = tiller.New(c)
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
		}

		validationResource, err = validation.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		// validationResource checks CRs for validation errors and sets the CR status.
		validationResource,

		// Following resources manage controller context information.
		appNamespaceResource,
		appcatalogResource,
		clientsResource,

		// Following resources bootstrap chart-operator in tenant clusters.
		tcNamespaceResource,
		tillerResource,
		chartOperatorResource,

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

	initCtxFunc := func(ctx context.Context, obj interface{}) (context.Context, error) {
		cc := controllercontext.Context{}
		ctx = controllercontext.NewContext(ctx, cc)

		return ctx, nil
	}

	var resourceSet *controller.ResourceSet
	{
		c := controller.ResourceSetConfig{
			InitCtx:   initCtxFunc,
			Logger:    config.Logger,
			Resources: resources,
		}

		resourceSet, err = controller.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resourceSet, nil
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