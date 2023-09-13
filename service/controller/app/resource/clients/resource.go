package clients

import (
	"context"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	kubeconfig "github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v6/service/internal/clientcache"
)

const (
	// Name is the identifier of the resource.
	Name = "clients"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	ClientCache           *clientcache.Resource
	HelmClient            helmclient.Interface
	HelmControllerBackend bool
	K8sClient             k8sclient.Interface
	Logger                micrologger.Logger
}

// Resource implements the clients resource.
type Resource struct {
	// Dependencies.
	clientCache           *clientcache.Resource
	helmClient            helmclient.Interface
	helmControllerBackend bool
	k8sClient             k8sclient.Interface
	logger                micrologger.Logger
}

// New creates a new configured clients resource.
func New(config Config) (*Resource, error) {
	if config.ClientCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClientCache must not be empty", config)
	}
	if config.HelmClient == helmclient.Interface(nil) {
		return nil, microerror.Maskf(invalidConfigError, "%T.HelmClient must not be empty", config)
	}
	if config.K8sClient == k8sclient.Interface(nil) {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		clientCache:           config.ClientCache,
		helmClient:            config.HelmClient,
		helmControllerBackend: config.HelmControllerBackend,
		k8sClient:             config.K8sClient,
		logger:                config.Logger,
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

		if !r.helmControllerBackend {
			r.logger.Debugf(
				ctx,
				"App CR %#q is management cluster app, re-using local clients",
				cr.Name,
			)

			return nil
		}

		// In case the Helm Controller backend is enabled, we also re-use the local clients
		// for the migration.
		r.logger.Debugf(
			ctx,
			"App CR %#q is management cluster app and has Helm Controller backend enabled, re-using local clients",
			cr.Name,
		)

		cc.MigrationClients = controllercontext.Clients{
			K8s:  r.k8sClient,
			Helm: r.helmClient,
		}

		return nil
	}

	r.logger.Debugf(ctx, "App CR %#q is workload cluster app, getting remote clients", cr.Name)

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
		r.logger.Debugf(ctx, "getting remote clients for %#q App CR has failed", cr.Name)

		return microerror.Mask(err)
	}

	// When Helm Controller Backend is enabled the primary clients are the
	// local clients. We however still need KubeConfig-based clients for
	// migration procedure which is removal of Chart CRs and related configuration.
	if r.helmControllerBackend {
		r.logger.Debugf(
			ctx,
			"App CR %#q has Helm Controller backend enabled, re-using local clients for operations and remote clients for migration",
			cr.Name,
		)

		cc.Clients = controllercontext.Clients{
			K8s:  r.k8sClient,
			Helm: r.helmClient,
		}

		cc.MigrationClients = controllercontext.Clients{
			K8s:  clients.K8sClient,
			Helm: clients.HelmClient,
		}
		r.logger.Debugf(ctx, "clients %#q", clients)

		return nil
	}

	r.logger.Debugf(
		ctx,
		"Remote clients for %#q App CR configured",
		cr.Name,
	)

	// If Helm Controller Backend is disabled we do not need migration clients,
	// and hence KubeConfig-based clients become the primary clients.
	cc.Clients = controllercontext.Clients{
		K8s:  clients.K8sClient,
		Helm: clients.HelmClient,
	}

	return nil
}
