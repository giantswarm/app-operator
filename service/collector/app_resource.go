package collector

import (
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"
)

// AppResourceConfig is this collector's configuration struct.
type AppResourceConfig struct {
	K8sClient kubernetes.Interface
	G8sClient versioned.Interface
	Logger    micrologger.Logger
}

// AppResource is the main struct for this collector.
type AppResource struct {
	k8sClient kubernetes.Interface
	g8sClient versioned.Interface
	logger    micrologger.Logger
}

// NewAppResource creates a new AppResource metrics collector
func NewAppResource(config AppResourceConfig) (*AppResource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	c := &AppResource{
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return c, nil
}

// Collect is the main metrics collection function.
func (c *AppResource) Collect(ch chan<- prometheus.Metric) error {
	// TODO
	return nil
}

// Describe emits the description for the metrics collected here.
func (c *AppResource) Describe(ch chan<- *prometheus.Desc) error {
	// TODO
	return nil
}
