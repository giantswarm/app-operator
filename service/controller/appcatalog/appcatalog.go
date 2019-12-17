package appcatalog

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/app-operator/pkg/project"
	v1 "github.com/giantswarm/app-operator/service/controller/appcatalog/v1"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
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
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			Name:      project.Name(),
			ResourceSets: []*controller.ResourceSet{
				resourceSetV1,
			},
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha1.AppCatalog)
			},
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
