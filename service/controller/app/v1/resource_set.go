package v1

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/service/controller/app/v1/appcatalog"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/chart"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/configmap"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/secret"
	"github.com/giantswarm/app-operator/service/controller/app/v1/resource/status"
)

// ResourceSetConfig contains necessary dependencies and settings for
// AppConfig controller ResourceSet configuration.
type ResourceSetConfig struct {
	// Dependencies.
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// Settings.
	ChartNamespace string
	ProjectName    string
	WatchNamespace string
}

// NewResourceSet returns a configured App controller ResourceSet.
func NewResourceSet(config ResourceSetConfig) (*controller.ResourceSet, error) {
	var err error

	// Dependencies.
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

	var kubeConfig kubeconfig.Interface
	{
		c := kubeconfig.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		kubeConfig, err = kubeconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appCatalog *appcatalog.AppCatalog
	{
		c := appcatalog.Config{
			G8sClient: config.G8sClient,
			Logger:    config.Logger,

			WatchNamespace: config.WatchNamespace,
		}

		appCatalog, err = appcatalog.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var chartResource controller.Resource
	{
		c := chart.Config{
			G8sClient: config.G8sClient,
			K8sClient: config.K8sClient,
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

	var configMapResource controller.Resource
	{
		c := configmap.Config{
			G8sClient: config.G8sClient,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			ChartNamespace: config.ChartNamespace,
			ProjectName:    config.ProjectName,
			WatchNamespace: config.WatchNamespace,
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
			G8sClient: config.G8sClient,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			ChartNamespace: config.ChartNamespace,
			ProjectName:    config.ProjectName,
			WatchNamespace: config.WatchNamespace,
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

	resources := []controller.Resource{
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
		cr, err := key.ToCustomResource(obj)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		catalogCR, err := appCatalog.GetCatalogForApp(ctx, cr)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		var k8sClient kubernetes.Interface
		var g8sClient versioned.Interface

		restConfig, err := kubeConfig.NewRESTConfigForApp(ctx, cr)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		g8sClient, err = versioned.NewForConfig(restConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		k8sClient, err = kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		c := controllercontext.Context{
			AppCatalog: *catalogCR,
			G8sClient:  g8sClient,
			K8sClient:  k8sClient,
		}
		ctx = controllercontext.NewContext(ctx, c)

		return ctx, nil
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
