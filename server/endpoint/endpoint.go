package endpoint

import (
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microendpoint/endpoint/healthz"
	"github.com/giantswarm/microendpoint/endpoint/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app-operator/v2/server/endpoint/status"
	"github.com/giantswarm/app-operator/v2/server/middleware"
	"github.com/giantswarm/app-operator/v2/service"
)

// Config represents the configuration used to construct an endpoint.
type Config struct {
	// Dependencies
	K8sClient  k8sclient.Interface
	Logger     micrologger.Logger
	Middleware *middleware.Middleware
	Service    *service.Service
}

// Endpoint is the endpoint collection.
type Endpoint struct {
	Healthz *healthz.Endpoint
	Status  *status.Endpoint
	Version *version.Endpoint
}

// New creates a new endpoint with given configuration.
func New(config Config) (*Endpoint, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Service == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Service or it's Healthz descendents must not be empty", config)
	}

	var err error

	var healthzEndpoint *healthz.Endpoint
	{
		c := healthz.Config{
			Logger: config.Logger,
		}

		healthzEndpoint, err = healthz.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var statusEndpoint *status.Endpoint
	{
		c := status.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		statusEndpoint, err = status.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var versionEndpoint *version.Endpoint
	{
		c := version.Config{
			Logger:  config.Logger,
			Service: config.Service.Version,
		}

		versionEndpoint, err = version.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	endpoint := &Endpoint{
		Healthz: healthzEndpoint,
		Status:  statusEndpoint,
		Version: versionEndpoint,
	}

	return endpoint, nil
}
