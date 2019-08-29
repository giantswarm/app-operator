package v1

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"github.com/spf13/afero"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/appcatalog"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/appnamespace"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/chart"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/chartoperator"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/clients"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/configmap"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/secret"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/status"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/tiller"
	"github.com/giantswarm/app-operator/service/controller/app/v1/values"
)

// ResourceSetConfig contains necessary dependencies and settings for
// AppConfig controller ResourceSet configuration.
type ResourceSetConfig struct {
	// Dependencies.
	FileSystem afero.Fs
	G8sClient  versioned.Interface
	K8sClient  kubernetes.Interface
	Logger     micrologger.Logger

	// Settings.
	ChartNamespace string
	ProjectName    string
	WatchNamespace string
}

// NewResourceSet returns a configured App controller ResourceSet.
func NewResourceSet(config ResourceSetConfig) (*controller.ResourceSet, error) {
	var err error

	// Dependencies.
	if config.FileSystem == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Fs must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
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
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ProjectName must not be empty", config)
	}

	var valuesService *values.Values
	{
		c := values.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		valuesService, err = values.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appcatalogResource controller.Resource
	{
		c := appcatalog.Config{
			G8sClient: config.G8sClient,
			Logger:    config.Logger,
		}
		appcatalogResource, err = appcatalog.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appNamespaceResource controller.Resource
	{
		c := appnamespace.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}
		appNamespaceResource, err = appnamespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var chartOperatorResource controller.Resource
	{
		c := chartoperator.Config{
			FileSystem: config.FileSystem,
			G8sClient:  config.G8sClient,
			K8sClient:  config.K8sClient,
			Logger:     config.Logger,
		}
		chartOperatorResource, err = chartoperator.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var chartResource controller.Resource
	{
		c := chart.Config{
			G8sClient: config.G8sClient,
			Logger:    config.Logger,

			ChartNamespace: config.ChartNamespace,
			ProjectName:    config.ProjectName,
			WatchNamespace: config.WatchNamespace,
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

	var clientsResource controller.Resource
	{
		c := clients.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		clientsResource, err = clients.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var configMapResource controller.Resource
	{
		c := configmap.Config{
			Logger: config.Logger,
			Values: valuesService,

			ChartNamespace: config.ChartNamespace,
			ProjectName:    config.ProjectName,
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

	var secretResource controller.Resource
	{
		c := secret.Config{
			Logger: config.Logger,
			Values: valuesService,

			ChartNamespace: config.ChartNamespace,
			ProjectName:    config.ProjectName,
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

	var statusResource controller.Resource
	{
		c := status.Config{
			G8sClient: config.G8sClient,
			Logger:    config.Logger,

			ChartNamespace: config.ChartNamespace,
		}

		statusResource, err = status.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var tillerResource controller.Resource
	{
		c := tiller.Config{
			Logger: config.Logger,
		}
		tillerResource, err = tiller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []controller.Resource{
		appNamespaceResource,
		appcatalogResource,
		clientsResource,
		tillerResource,
		chartOperatorResource,
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

		if key.VersionLabel(cr) == VersionBundle().Version {
			return true
		}

		return false
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

func toCRUDResource(logger micrologger.Logger, ops controller.CRUDResourceOps) (*controller.CRUDResource, error) {
	c := controller.CRUDResourceConfig{
		Logger: logger,
		Ops:    ops,
	}

	r, err := controller.NewCRUDResource(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r, nil
}
