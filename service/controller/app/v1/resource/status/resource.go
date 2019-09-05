package status

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

const (
	Name = "statusv1"
)

// Config represents the configuration used to create a new chartstatus resource.
type Config struct {
	G8sClient versioned.Interface
	Logger    micrologger.Logger

	ChartNamespace string
}

// Resource implements the chartstatus resource.
type Resource struct {
	g8sClient versioned.Interface
	logger    micrologger.Logger

	chartNamespace string
}

func New(config Config) (*Resource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		g8sClient: config.G8sClient,
		logger:    config.Logger,

		chartNamespace: config.ChartNamespace,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

// equals asseses the equality of AppStatuses with regards to distinguishing
// fields.
func equals(a, b v1alpha1.AppStatus) bool {
	if a.AppVersion != b.AppVersion {
		return false
	}

	releaseA := key.StatusRelease(a)
	releaseB := key.StatusRelease(b)

	if releaseA.LastDeployed != releaseB.LastDeployed {
		return false
	}
	if releaseA.Status != releaseB.Status {
		return false
	}
	if a.Version != b.Version {
		return false
	}

	return true
}
