package catalog

import (
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v6/pkg/controller"
	"github.com/giantswarm/operatorkit/v6/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-operator/v5/pkg/project"
)

const catalogControllerSuffix = "-catalog"

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	MaxEntriesPerApp int
	Provider         string
	UniqueApp        bool
}

type Catalog struct {
	*controller.Controller
}

func NewCatalog(config Config) (*Catalog, error) {
	var err error

	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var resources []resource.Interface
	{
		c := catalogResourcesConfig(config)
		resources, err = newCatalogResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var catalogController *controller.Controller
	{
		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			Resources: resources,
			NewRuntimeObjectFunc: func() client.Object {
				return new(v1alpha1.Catalog)
			},

			Name: project.Name() + catalogControllerSuffix,
		}

		catalogController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &Catalog{
		Controller: catalogController,
	}

	return c, nil
}
