package app

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/crud"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"github.com/spf13/afero"

	"github.com/giantswarm/app-operator/pkg/project"
	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/key"
	"github.com/giantswarm/app-operator/service/controller/app/resource/appcatalog"
	"github.com/giantswarm/app-operator/service/controller/app/resource/appnamespace"
	"github.com/giantswarm/app-operator/service/controller/app/resource/chart"
	"github.com/giantswarm/app-operator/service/controller/app/resource/chartoperator"
	"github.com/giantswarm/app-operator/service/controller/app/resource/clients"
	"github.com/giantswarm/app-operator/service/controller/app/resource/configmap"
	"github.com/giantswarm/app-operator/service/controller/app/resource/releasemigration"
	"github.com/giantswarm/app-operator/service/controller/app/resource/secret"
	"github.com/giantswarm/app-operator/service/controller/app/resource/status"
	"github.com/giantswarm/app-operator/service/controller/app/resource/tcnamespace"
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
	ChartNamespace    string
	HTTPClientTimeout time.Duration
	ImageRegistry     string
	UniqueApp         bool
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
	if config.HTTPClientTimeout == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.HTTPClientTimeout must not be empty", config)
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
			G8sClient: config.K8sClient.G8sClient(),
			Logger:    config.Logger,

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

			HTTPClientTimeout: config.HTTPClientTimeout,
			ImageRegistry:     config.ImageRegistry,
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

	resources := []resource.Interface{
		// Following resources manage controller context information.
		appNamespaceResource,
		appcatalogResource,
		clientsResource,

		// Following resources bootstrap chart-operator in tenant clusters.
		tcNamespaceResource,
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

	handlesFunc := func(obj interface{}) bool {
		cr, err := key.ToCustomResource(obj)
		if err != nil {
			return false
		}

		fmt.Printf("UNIQUE APP %t", config.UniqueApp)
		fmt.Printf("CR NAME %#q", cr.Name)

		cpRes := key.VersionLabel(cr) == project.AppControlPlaneVersion()
		fmt.Printf("MATCHING CP VERSION %t", cpRes)

		tcRes := key.VersionLabel(cr) == project.AppTenantVersion()
		fmt.Printf("MATCHING CP VERSION %t", tcRes)

		// When app-operator is deployed as a unique app it only processes
		// control plane app CRs. These CRs always have the version label
		// app-operator.giantswarm.io/verson: 0.0.0
		if config.UniqueApp {
			return key.VersionLabel(cr) == project.AppControlPlaneVersion()
		}

		// Currently tenant cluster app CRs always have the version label
		// app-operator.giantswarm.io/verson: 1.0.0
		// This hardcoding will be removed in a future release and we will then
		// use project.Version().
		return key.VersionLabel(cr) == project.AppTenantVersion()
	}

	initCtxFunc := func(ctx context.Context, obj interface{}) (context.Context, error) {
		cc := controllercontext.Context{}
		ctx = controllercontext.NewContext(ctx, cc)

		return ctx, nil
	}

	var resourceSet *controller.ResourceSet
	{
		c := controller.ResourceSetConfig{
			Handles:   handlesFunc,
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
