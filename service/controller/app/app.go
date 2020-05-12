package app

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/app-operator/pkg/project"
	"github.com/giantswarm/app-operator/service/controller/app/key"
)

type Config struct {
	Fs        afero.Fs
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	ChartNamespace string
	ImageRegistry  string
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

	if config.ImageRegistry == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ImageRegistry must not be empty", config)
	}

	var resourceSetV1 *controller.ResourceSet
	{
		c := ResourceSetConfig{
			ChartNamespace: config.ChartNamespace,
			FileSystem:     config.Fs,
			ImageRegistry:  config.ImageRegistry,
			K8sClient:      config.K8sClient,
			Logger:         config.Logger,
		}

		resourceSetV1, err = NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appController *controller.Controller
	{
		c := controller.Config{
			CRD:       v1alpha1.NewAppCRD(),
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			Name:      project.Name(),
			ResourceSets: []*controller.ResourceSet{
				resourceSetV1,
			},
			Selector: key.LabelSelectorService(),
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha1.App)
			},
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
