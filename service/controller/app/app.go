package app

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/controller"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/app-operator/v2/pkg/label"
	"github.com/giantswarm/app-operator/v2/pkg/project"
	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
)

type Config struct {
	Fs        afero.Fs
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	ChartNamespace    string
	HTTPClientTimeout time.Duration
	ImageRegistry     string
	PauseAnnotation   string
	UniqueApp         bool
	WebhookAuthToken  string
	WebhookBaseURL    string
}

type App struct {
	*controller.Controller
}

func NewApp(config Config) (*App, error) {
	var err error

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
	if config.WebhookBaseURL == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.WebhookBaseURL not be empty", config)
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
			FileSystem: config.Fs,
			K8sClient:  config.K8sClient,
			Logger:     config.Logger,

			ChartNamespace:    config.ChartNamespace,
			HTTPClientTimeout: config.HTTPClientTimeout,
			ImageRegistry:     config.ImageRegistry,
			UniqueApp:         config.UniqueApp,
			WebhookAuthToken:  config.WebhookAuthToken,
			WebhookBaseURL:    config.WebhookBaseURL,
		}

		resources, err = newAppResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appController *controller.Controller
	{
		var pause map[string]string
		if config.PauseAnnotation != "" {
			pause = map[string]string{
				config.PauseAnnotation: "true",
			}
		}

		c := controller.Config{
			InitCtx:   initCtxFunc,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			Pause:     pause,
			Resources: resources,
			Selector:  label.AppVersionSelector(config.UniqueApp),
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha1.App)
			},

			Name: project.Name(),
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
