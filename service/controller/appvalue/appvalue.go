package appvalue

import (
	"context"
	"sync"

	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type AppValueConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type AppValue struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	apps sync.Map
}

func NewAppValue(config AppValueConfig) (*AppValue, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	c := &AppValue{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		apps: sync.Map{},
	}

	return c, nil
}

func (c *AppValue) Boot(ctx context.Context) {
	err := c.buildCache(ctx)
	if err != nil {
		panic(err)
	}
}
