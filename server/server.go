package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/giantswarm/microerror"
	microserver "github.com/giantswarm/microkit/server"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/viper"

	"github.com/giantswarm/app-operator/v7/pkg/project"
	"github.com/giantswarm/app-operator/v7/server/endpoint"
	"github.com/giantswarm/app-operator/v7/service"
)

// Config represents the configuration used to construct server object.
type Config struct {
	Logger  micrologger.Logger
	Service *service.Service

	Viper            *viper.Viper
	WebhookAuthToken string
}

// New creates a new server object with given configuration.
func New(config Config) (microserver.Server, error) {
	var err error

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Service == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Service must not be empty", config)
	}

	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Viper must not be empty", config)
	}

	var endpointCollection *endpoint.Endpoint
	{
		c := endpoint.Config{
			Logger:  config.Logger,
			Service: config.Service,
		}

		endpointCollection, err = endpoint.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	newServer := &server{
		// Dependencies
		logger: config.Logger,

		// Internals
		bootOnce: sync.Once{},
		config: microserver.Config{
			Logger:      config.Logger,
			ServiceName: project.Name(),
			Viper:       config.Viper,
			Endpoints: []microserver.Endpoint{
				endpointCollection.Healthz,
				endpointCollection.Version,
			},
			ErrorEncoder: errorEncoder,
		},
		shutdownOnce: sync.Once{},
	}

	return newServer, nil
}

type server struct {
	// Dependencies
	logger micrologger.Logger

	// Internals
	bootOnce     sync.Once
	config       microserver.Config
	shutdownOnce sync.Once
}

func (s *server) Boot() {
	s.bootOnce.Do(func() {
		// Insert here custom boot logic for server/endpoint/middleware if needed.
	})
}

func (s *server) Config() microserver.Config {
	return s.config
}

func (s *server) Shutdown() {
	s.shutdownOnce.Do(func() {
		// Insert here custom shutdown logic for server/endpoint/middleware if needed.
	})
}

func errorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
	rErr := err.(microserver.ResponseError)
	uErr := rErr.Underlying()

	rErr.SetCode(microserver.CodeInternalError)
	rErr.SetMessage(uErr.Error())
	w.WriteHeader(http.StatusInternalServerError)
}
