package configmap

import (
	"context"
	"sync"

	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/app-operator/v2/pkg/label"
)

type AppValueConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	UniqueApp bool
}

type AppValue struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	configMapToApps sync.Map
	selector        labels.Selector
	unique          bool
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

		configMapToApps: sync.Map{},
		selector:        label.AppVersionSelector(config.UniqueApp),
		unique:          config.UniqueApp,
	}

	return c, nil
}

func (c *AppValue) Boot(ctx context.Context) {
	// Watching configmap's changes.
	go c.watch(ctx)

	// Building a cache of configmaps and link each app to configmaps.
	err := c.buildCache(ctx)
	if err != nil {
		panic(err)
	}
}
