package clients

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/key"
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
	ImageRegistry string
}

// Resource implements the clients resource.
type Resource struct {
	// Dependencies.
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	// Settings.
	imageRegistry string
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

	r := &Resource{
		// Dependencies.
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		// Settings
		imageRegistry: config.ImageRegistry,
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

	restConfig, err := kubeConfig.NewRESTConfigForApp(ctx, cr)
	if kubeconfig.IsNotFoundError(err) {
		// Set status so we don't try to connect to the tenant cluster
		// again in this reconciliation loop.
		cc.Status.TenantCluster.IsUnavailable = true

		r.logger.LogCtx(ctx, "level", "debug", "message", "kubeconfig secret not found")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil

	} else if err != nil {
		return microerror.Mask(err)
	}

	var g8sClient versioned.Interface
	{
		g8sClient, err = versioned.NewForConfig(restConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var k8sClient kubernetes.Interface
	{
		k8sClient, err = kubernetes.NewForConfig(restConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var tillerNamespace string
	{
		// When InCluster is used we use the Tiller deployed in kube-system.
		if key.InCluster(cr) {
			tillerNamespace = metav1.NamespaceSystem
		} else {
			tillerNamespace = "giantswarm"
		}
	}

	var helmClient helmclient.Interface
	{
		c := helmclient.Config{
			K8sClient: k8sClient,
			Logger:    r.logger,

			EnsureTillerInstalledMaxWait: 30 * time.Second,
			RestConfig:                   restConfig,
			TillerImageRegistry:          r.imageRegistry,
			TillerNamespace:              tillerNamespace,
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	cc.Clients = controllercontext.Clients{
		G8s:  g8sClient,
		K8s:  k8sClient,
		Helm: helmClient,
	}

	return nil
}
