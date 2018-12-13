package service

import (
	"github.com/spf13/viper"
	"sync"

	"github.com/giantswarm/app-operator/flag"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

// Config represents the configuration used to create a new service.
type Config struct {
	Logger micrologger.Logger
	Flag   *flag.Flag
	Viper  *viper.Viper

	Description string
	GitCommit   string
	Name        string
	Source      string
}

// New creates a new service with given configuration.
func New(config Config) (*Service, error) {
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Flag must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Viper must not be empty", config)
	}

	newService := &Service{
		// Internals
		bootOnce: sync.Once{},
	}

	return newService, nil
}

// Service is a type providing implementation of microkit service interface.
type Service struct {
	// Internals
	bootOnce sync.Once
}

// Boot starts top level service implementation.
func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		// Insert service startup logic here.
	})
}
