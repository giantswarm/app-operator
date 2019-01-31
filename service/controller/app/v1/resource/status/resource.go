package status

import (
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
)

const (
	Name = "chartstatusv1"
)

// Config represents the configuration used to create a new chartstatus resource.
type Config struct {
	G8sClient  versioned.Interface
	K8sClient  kubernetes.Interface
	KubeConfig kubeconfig.Interface
	Logger     micrologger.Logger

	WatchNamespace string
}

// Resource implements the chartstatus resource.
type Resource struct {
	g8sClient  versioned.Interface
	k8sClient  kubernetes.Interface
	kubeConfig kubeconfig.Interface
	logger     micrologger.Logger

	watchNamespace string
}

func New(config Config) (*Resource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.KubeConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.KubeConfig must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.WatchNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.WatchNamespace must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		g8sClient:  config.G8sClient,
		kubeConfig: config.KubeConfig,
		k8sClient:  config.K8sClient,
		logger:     config.Logger,

		watchNamespace: config.WatchNamespace,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}
