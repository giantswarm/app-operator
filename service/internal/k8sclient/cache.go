package k8sclient

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	gocache "github.com/patrickmn/go-cache"
	"github.com/spf13/afero"
	"k8s.io/client-go/rest"
)

const (
	expiration = 1 * time.Hour
)

type Config struct {
	// Dependencies.
	Fs        afero.Fs
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	// Settings.
	HTTPClientTimeout time.Duration
}

type Resource struct {
	// Dependencies.
	cache     *gocache.Cache
	fs        afero.Fs
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	// Settings.
	httpClientTimeout time.Duration
}

// New creates a new configured clients resource.
func New(config Config) (*Resource, error) {
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

	r := &Resource{
		// Dependencies.
		cache:     gocache.New(expiration, expiration/2),
		fs:        config.Fs,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		// Settings
		httpClientTimeout: config.HTTPClientTimeout,
	}

	return r, nil
}

func (r *Resource) GetK8sClient(ctx context.Context, kubeConfig *v1alpha1.AppSpecKubeConfig) (k8sclient.Interface, error) {
	k := fmt.Sprintf("%s/%s", kubeConfig.Secret.Namespace, kubeConfig.Secret.Name)

	fmt.Println("CHECKING CACHE")
	if v, ok := r.cache.Get(k); ok {
		clients, ok := v.(k8sclient.Interface)
		if !ok {
			return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", k8sclient.Clients{}, v)
		}

		return clients, nil
	}

	fmt.Println("CACHE FAULT")
	fmt.Println("CREATE CLIENT")
	clients, err := r.generateK8sClient(ctx, kubeConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	fmt.Println("CREATE KEY")
	r.cache.SetDefault(k, clients)

	fmt.Println("RETURNING")
	return clients, nil
}

func (r *Resource) generateK8sClient(ctx context.Context, config *v1alpha1.AppSpecKubeConfig) (k8sclient.Interface, error) {
	var err error

	var kubeConfig kubeconfig.Interface
	{
		c := kubeconfig.Config{
			K8sClient: r.k8sClient.K8sClient(),
			Logger:    r.logger,
		}

		kubeConfig, err = kubeconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var restConfig *rest.Config
	{
		restConfig, err = kubeConfig.NewRESTConfigForApp(ctx, config.Secret.Name, config.Secret.Namespace)
		if kubeconfig.IsNotFoundError(err) {
			return nil, microerror.Mask(err)
		} else if err != nil {
			return nil, microerror.Mask(err)
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
			return nil, microerror.Mask(err)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return k8sClient, nil
}
