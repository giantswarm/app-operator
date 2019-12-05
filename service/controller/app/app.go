package app

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/informer"
	"github.com/spf13/afero"

	"github.com/giantswarm/app-operator/pkg/project"
	v1 "github.com/giantswarm/app-operator/service/controller/app/v1"
)

type Config struct {
	Fs        afero.Fs
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	ChartNamespace string
	ImageRegistry  string
	WatchNamespace string
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

	var crdClient *k8scrdclient.CRDClient
	{
		c := k8scrdclient.Config{
			K8sExtClient: config.K8sClient.ExtClient(),
			Logger:       config.Logger,
		}

		crdClient, err = k8scrdclient.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var newInformer *informer.Informer
	{
		c := informer.Config{
			Logger:  config.Logger,
			Watcher: config.K8sClient.G8sClient().ApplicationV1alpha1().Apps(config.WatchNamespace),

			RateWait:     informer.DefaultRateWait,
			ResyncPeriod: informer.DefaultResyncPeriod,
		}

		newInformer, err = informer.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resourceSetV1 *controller.ResourceSet
	{
		c := v1.ResourceSetConfig{
			ChartNamespace: config.ChartNamespace,
			FileSystem:     config.Fs,
			ImageRegistry:  config.ImageRegistry,
			K8sClient:      config.K8sClient,
			Logger:         config.Logger,
			WatchNamespace: config.WatchNamespace,
		}

		resourceSetV1, err = v1.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appController *controller.Controller
	{
		c := controller.Config{
			CRD:       v1alpha1.NewAppCRD(),
			CRDClient: crdClient,
			Informer:  newInformer,
			Logger:    config.Logger,
			ResourceSets: []*controller.ResourceSet{
				resourceSetV1,
			},
			RESTClient: config.K8sClient.G8sClient().CoreV1alpha1().RESTClient(),

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
