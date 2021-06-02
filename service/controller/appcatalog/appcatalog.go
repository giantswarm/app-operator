package appcatalog

import (
	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/controller"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/app-operator/v4/pkg/project"
)

const appCatalogControllerSuffix = "-appcatalog"

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	MaxEntriesPerApp int
	UniqueApp        bool
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

	var resources []resource.Interface
	{
		c := appCatalogResourcesConfig(config)
		resources, err = newAppCatalogResources(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appCatalogController *controller.Controller
	{
		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			Resources: resources,
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha1.AppCatalog)
			},

			Name: project.Name() + appCatalogControllerSuffix,
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
