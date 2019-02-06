package chartcrd

import (
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
)

const (
	Name = "chartcrdv1"
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
	logger micrologger.Logger

	watchNamespace string
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.WatchNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.WatchNamespace must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		logger: config.Logger,

		watchNamespace: config.WatchNamespace,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}
