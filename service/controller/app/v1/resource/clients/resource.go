package clients

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
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

	// Settings.
	ImageRegistry   string
	TillerNamespace string
}

// Resource implements the clients resource.
type Resource struct {
	// Dependencies.
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	// Settings.
	imageRegistry   string
	tillerNamespace string
}

// New creates a new configured clients resource.
func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ImageRegistry == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ImageRegistry must not be empty", config)
	}
	if config.TillerNamespace == "" {
		config.TillerNamespace = "giantswarm"
	}

	r := &Resource{
		// Dependencies.
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		// Settings
		imageRegistry:   config.ImageRegistry,
		tillerNamespace: config.TillerNamespace,
	}

	return r, nil
}

func (*Resource) Name() string {
	return Name
}

// addClientsToContext adds g8s and k8s clients based on the kubeconfig
// settings for the app CR.
func (r *Resource) addClientsToContext(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if cc.Status.TenantCluster.IsDeleting {
		return nil
	}

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

	var restConfig *rest.Config
	{
		restConfig, err = kubeConfig.NewRESTConfigForApp(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var k8sClient k8sclient.Interface
	{
		c := k8sclient.ClientsConfig{
			Logger: r.logger,
			SchemeBuilder: k8sclient.SchemeBuilder{
				// Add application scheme.
				v1alpha1.AddToScheme,
			},

			RestConfig: restConfig,
		}

		k8sClient, err = k8sclient.NewClients(c)
		if err != nil {
			return microerror.Mask(err)
		}

		cc.K8sClient = k8sClient
	}

	{
		c := helmclient.Config{
			K8sClient: k8sClient.K8sClient(),
			Logger:    r.logger,

			EnsureTillerInstalledMaxWait: 30 * time.Second,
			RestConfig:                   restConfig,
			TillerImageRegistry:          r.imageRegistry,
			TillerNamespace:              r.tillerNamespace,
		}

		helmClient, err := helmclient.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
		cc.HelmClient = helmClient
	}

	return nil
}
