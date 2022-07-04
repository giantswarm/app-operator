package clients

import (
	"context"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclient"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app-operator/v7/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v7/service/internal/clientcache"
)

const (
	// Name is the identifier of the resource.
	Name = "clients"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	ClientCache *clientcache.Resource
	HelmClient  helmclient.Interface
	K8sClient   k8sclient.Interface
	Logger      micrologger.Logger
}

// Resource implements the clients resource.
type Resource struct {
	// Dependencies.
	clientCache *clientcache.Resource
	helmClient  helmclient.Interface
	k8sClient   k8sclient.Interface
	logger      micrologger.Logger
}

// New creates a new configured clients resource.
func New(config Config) (*Resource, error) {
	if config.ClientCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClientCache must not be empty", config)
	}
	if config.HelmClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HelmClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		clientCache: config.ClientCache,
		helmClient:  config.HelmClient,
		k8sClient:   config.K8sClient,
		logger:      config.Logger,
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

	// App CR uses inCluster so we can reuse the existing clients.
	if key.InCluster(cr) {
		cc.Clients = controllercontext.Clients{
			K8s:  r.k8sClient,
			Helm: r.helmClient,
		}

		return nil
	}

	clients, err := r.clientCache.GetClients(ctx, &cr.Spec.KubeConfig)
	if kubeconfig.IsNotFoundError(err) {
		// Set status so we don't try to connect to the workload cluster
		// again in this reconciliation loop.
		cc.Status.ClusterStatus.IsUnavailable = true

		r.logger.Debugf(ctx, "kubeconfig secret not found")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if tenant.IsAPINotAvailable(err) {
		// Set status so we don't try to connect to the workload cluster
		// again in this reconciliation loop.
		cc.Status.ClusterStatus.IsUnavailable = true

		r.logger.Debugf(ctx, "workload API not available yet")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	cc.Clients = controllercontext.Clients{
		K8s:  clients.K8sClient,
		Helm: clients.HelmClient,
	}

	return nil
}
