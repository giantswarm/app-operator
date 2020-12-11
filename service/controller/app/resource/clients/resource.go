package clients

import (
	"context"
	"github.com/giantswarm/app/v4/pkg/key"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "clients"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// Settings.
	HTTPClientTimeout time.Duration
	ImageRegistry     string
}

// Resource implements the clients resource.
type Resource struct {
	// Dependencies.
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	// Settings.
	httpClientTimeout time.Duration
	imageRegistry     string
}

// New creates a new configured clients resource.
func New(config Config) (*Resource, error) {
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

	r := &Resource{
		// Dependencies.
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		// Settings
		httpClientTimeout: config.HTTPClientTimeout,
		imageRegistry:     config.ImageRegistry,
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

	if cc.Status.ClusterStatus.IsDeleting {
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
		if key.InCluster(cr) {
			restConfig, err = rest.InClusterConfig()
			if err != nil {
				return microerror.Mask(err)
			}
		} else {
			restConfig, err = kubeConfig.NewRESTConfigForApp(ctx, key.KubeConfigSecretName(cr), key.KubeConfigSecretNamespace(cr))
			if kubeconfig.IsNotFoundError(err) {
				// Set status so we don't try to connect to the tenant cluster
				// again in this reconciliation loop.
				cc.Status.ClusterStatus.IsUnavailable = true

				r.logger.Debugf(ctx, "kubeconfig secret not found")
				r.logger.Debugf(ctx, "canceling resource")
				return nil

			} else if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	var k8sClient k8sclient.Interface
	{
		c := k8sclient.ClientsConfig{
			Logger:     r.logger,
			RestConfig: rest.CopyConfig(restConfig),
		}

		k8sClient, err = k8sclient.NewClients(c)
		if tenant.IsAPINotAvailable(err) {
			// Set status so we don't try to connect to the tenant cluster
			// again in this reconciliation loop.
			cc.Status.ClusterStatus.IsUnavailable = true

			r.logger.Debugf(ctx, "tenant API not available yet")
			r.logger.Debugf(ctx, "canceling resource")
			return nil

		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	fs := afero.NewOsFs()

	var helmClient helmclient.Interface
	{
		c := helmclient.Config{
			Fs:         fs,
			K8sClient:  k8sClient.K8sClient(),
			Logger:     r.logger,
			RestClient: k8sClient.RESTClient(),
			RestConfig: k8sClient.RESTConfig(),

			HTTPClientTimeout: r.httpClientTimeout,
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	cc.Clients = controllercontext.Clients{
		K8s:  k8sClient,
		Helm: helmClient,
	}

	return nil
}
