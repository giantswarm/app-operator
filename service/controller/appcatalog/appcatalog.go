package appcatalog

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/informer"

	"github.com/giantswarm/app-operator/pkg/project"
	v1 "github.com/giantswarm/app-operator/service/controller/appcatalog/v1"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	WatchNamespace string
}

type AppCatalog struct {
	*controller.Controller
}

func NewAppCatalog(config Config) (*AppCatalog, error) {
	var err error

	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
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
			Watcher: config.K8sClient.G8sClient().ApplicationV1alpha1().AppCatalogs(),

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
			Logger: config.Logger,
		}

		resourceSetV1, err = v1.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appCatalogController *controller.Controller
	{
		c := controller.Config{
			CRD:       v1alpha1.NewAppCatalogCRD(),
			CRDClient: crdClient,
			Informer:  newInformer,
			Logger:    config.Logger,
			ResourceSets: []*controller.ResourceSet{
				resourceSetV1,
			},
			RESTClient: config.K8sClient.RESTClient(),

			Name: project.Name(),
		}

		appCatalogController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &AppCatalog{
		Controller: appCatalogController,
	}

	return c, nil
}
