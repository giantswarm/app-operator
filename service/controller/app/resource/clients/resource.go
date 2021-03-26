package clients

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app-operator/v4/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v4/service/internal/k8sclientcache"
)

const (
	// Name is the identifier of the resource.
	Name = "clients"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	HelmClient     helmclient.Interface
	K8sClient      k8sclient.Interface
	K8sClientCache *k8sclientcache.Resource
	Logger         micrologger.Logger
}

// Resource implements the clients resource.
type Resource struct {
	// Dependencies.
	helmClient     helmclient.Interface
	k8sClient      k8sclient.Interface
	k8sClientCache *k8sclientcache.Resource
	logger         micrologger.Logger
}

// New creates a new configured clients resource.
func New(config Config) (*Resource, error) {
	if config.HelmClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HelmClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.K8sClientCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CachedK8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		helmClient:     config.HelmClient,
		k8sClient:      config.K8sClient,
		k8sClientCache: config.K8sClientCache,
		logger:         config.Logger,
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

	clients, err := r.k8sClientCache.GetClients(ctx, &cr.Spec.KubeConfig)
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
