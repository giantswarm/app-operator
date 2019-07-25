package clients

import (
	"context"
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

const (
	// Name is the identifier of the resource.
	Name = "clientsv1"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	// Dependencies.
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

// New creates a new configured clients resource.
func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return r, nil
}

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.addClientsToContext(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.addClientsToContext(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (*Resource) Name() string {
	return Name
}

func (r *Resource) addClientsToContext(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	var k8sClient kubernetes.Interface
	var g8sClient versioned.Interface

	if !cc.Status.TenantCluster.IsDeleting {

		var kubeConfig kubeconfig.Interface
		{
			c := kubeconfig.Config{
				K8sClient: r.k8sClient,
				Logger:    r.logger,
			}

			kubeConfig, err = kubeconfig.New(c)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		restConfig, err := kubeConfig.NewRESTConfigForApp(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}

		if cc.G8sClient == nil {
			g8sClient, err = versioned.NewForConfig(restConfig)
			if err != nil {
				return microerror.Mask(err)
			}
			cc.G8sClient = g8sClient
		}

		if cc.K8sClient == nil {
			k8sClient, err = kubernetes.NewForConfig(restConfig)
			if err != nil {
				return microerror.Mask(err)
			}
			cc.K8sClient = k8sClient
		}
	}
	return nil
}
