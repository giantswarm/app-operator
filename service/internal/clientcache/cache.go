package clientcache

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclient"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	gocache "github.com/patrickmn/go-cache"
	"github.com/spf13/afero"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	expiration = 10 * time.Minute
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

type clients struct {
	K8sClient  k8sclient.Interface
	HelmClient helmclient.Interface
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

func (r *Resource) GetClients(ctx context.Context, kubeConfig *v1alpha1.AppSpecKubeConfig) (*clients, error) {
	k := fmt.Sprintf("%s/%s", kubeConfig.Secret.Namespace, kubeConfig.Secret.Name)

	if v, ok := r.cache.Get(k); ok {
		c, ok := v.(clients)
		if !ok {
			return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", clients{}, v)
		}

		return &c, nil
	}

	var c clients
	{
		k8sClient, err := r.generateK8sClient(ctx, kubeConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		helmClient, err := r.generateHelmClient(k8sClient)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		c = clients{
			K8sClient:  k8sClient,
			HelmClient: helmClient,
		}
	}

	r.cache.SetDefault(k, c)

	return &c, nil
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

func (r *Resource) generateHelmClient(k8sClient k8sclient.Interface) (helmclient.Interface, error) {
	var helmClient *helmclient.Client
	{
		restMapper, err := apiutil.NewDynamicRESTMapper(rest.CopyConfig(k8sClient.RESTConfig()))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		c := helmclient.Config{
			Fs:         r.fs,
			K8sClient:  k8sClient.K8sClient(),
			Logger:     r.logger,
			RestClient: k8sClient.RESTClient(),
			RestConfig: k8sClient.RESTConfig(),
			RestMapper: restMapper,

			HTTPClientTimeout: r.httpClientTimeout,
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return helmClient, nil
}
