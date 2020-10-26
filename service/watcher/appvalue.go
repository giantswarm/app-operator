package watcher

import (
	"context"
	"sync"

	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/app-operator/v2/pkg/label"
)

type AppValueWatcherConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	UniqueApp bool
}

type AppValueWatcher struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	resourcesToApps sync.Map
	selector        labels.Selector
	unique          bool
}

func NewAppValueWatcher(config AppValueWatcherConfig) (*AppValueWatcher, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	c := &AppValueWatcher{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		resourcesToApps: sync.Map{},
		selector:        label.AppVersionSelector(config.UniqueApp),
		unique:          config.UniqueApp,
	}

	return c, nil
}

func (c *AppValueWatcher) Boot(ctx context.Context) {
	// Watch for configmap changes.
	go c.watchConfigMap(ctx)

	go c.watchSecret(ctx)

	// Build a cache of configmaps and link each app to its configmaps.
	go c.buildCache(ctx)
}
