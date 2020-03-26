package kubeconfig

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
)

// Config represents the configuration used to create a new kubeconfig library
// instance.
type Config struct {
	Logger    micrologger.Logger
	K8sClient kubernetes.Interface
}

func (c *Config) Validate() error {
	if c.Logger == nil {
		return microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", c)
	}
	if c.K8sClient == nil {
		return microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", c)
	}
	return nil
}
