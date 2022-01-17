package app

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v6/pkg/controller"
	"github.com/giantswarm/operatorkit/v6/pkg/resource"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-operator/v5/pkg/label"
	"github.com/giantswarm/app-operator/v5/pkg/project"
	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v5/service/internal/clientcache"
)

const appControllerSuffix = "-app"

type Config struct {
	Fs          afero.Fs
	K8sClient   k8sclient.Interface
	ClientCache *clientcache.Resource
	Logger      micrologger.Logger

	ChartNamespace    string
	HTTPClientTimeout time.Duration
	ImageRegistry     string
	PodNamespace      string
	Provider          string
	ResyncPeriod      time.Duration
	UniqueApp         bool
	WatchNamespace    string
	WorkloadClusterID string
}

type App struct {
	*controller.Controller
}

func NewApp(config Config) (*App, error) {
	var err error

	if config.ClientCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClientCache must not be empty", config)
	}
	if config.Fs == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Fs must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.HTTPClientTimeout == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.HTTPClientTimeout must not be empty", config)
	}
	if config.ImageRegistry == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ImageRegistry must not be empty", config)
	}
	if config.PodNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.PodNamespace must not be empty", config)
	}
	if config.Provider == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}
	if config.ResyncPeriod == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.ResyncPeriod must not be empty", config)
	}

	// For non-unique instances the watch namespace and cluster ID must
	// provided so the correct app CRs are watched
	if !config.UniqueApp {
		if config.WatchNamespace == "" {
			return nil, microerror.Maskf(invalidConfigError, "%T.WatchNamespace must not be empty for non-unique instance", config)
		}
		if config.WorkloadClusterID == "" {
			return nil, microerror.Maskf(invalidConfigError, "%T.WorkloadClusterID must not be empty for non-unique instance", config)
		}
	}

	// TODO: Remove usage of deprecated controller context.
	//
	//	https://github.com/giantswarm/giantswarm/issues/12324
	//
	initCtxFunc := func(ctx context.Context, obj interface{}) (context.Context, error) {
		cc := controllercontext.Context{}
		ctx = controllercontext.NewContext(ctx, cc)

		return ctx, nil
	}

	var resources []resource.Interface
	{
		c := appResourcesConfig{
			ClientCache: config.ClientCache,
			FileSystem:  config.Fs,
			K8sClient:   config.K8sClient,
			Logger:      config.Logger,

			ChartNamespace:    config.ChartNamespace,
			HTTPClientTimeout: config.HTTPClientTimeout,
			ImageRegistry:     config.ImageRegistry,
			ProjectName:       project.Name(),
			Provider:          config.Provider,
			UniqueApp:         config.UniqueApp,
			WorkloadClusterID: config.WorkloadClusterID,
		}

		resources, err = newAppResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appController *controller.Controller
	{
		c := controller.Config{
			InitCtx:      initCtxFunc,
			K8sClient:    config.K8sClient,
			Logger:       config.Logger,
			ResyncPeriod: config.ResyncPeriod,
			Pause: map[string]string{
				annotation.AppOperatorPaused: "true",
			},
			Resources: resources,
			Selector:  label.ClusterSelector(config.WorkloadClusterID),
			NewRuntimeObjectFunc: func() client.Object {
				return new(v1alpha1.App)
			},

			Name: project.Name() + appControllerSuffix,
		}

		if !config.UniqueApp {
			c.Namespace = config.WatchNamespace
		}

		appController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &App{
		Controller: appController,
	}

	return c, nil
}
