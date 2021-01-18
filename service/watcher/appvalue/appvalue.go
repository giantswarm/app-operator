package appvalue

import (
	"context"
	"sync"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/app-operator/v3/pkg/label"
)

type AppValueConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	PodNamespace string
	UniqueApp    bool
}

type AppValue struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	appNamespace    string
	resourcesToApps sync.Map
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

	if config.PodNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.PodNamespace must not be empty", config)
	}

	var appNamespace string

	if !config.UniqueApp {
		// Only populate the cache for app CRs in the current namespace. The
		// label selector excludes the operator's own app CR which has the
		// unique version.
		appNamespace = config.PodNamespace
	}

	c := &AppValue{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		appNamespace:    appNamespace,
		resourcesToApps: sync.Map{},
		selector:        label.AppVersionSelector(config.UniqueApp),
		unique:          config.UniqueApp,
	}

	return c, nil
}

func (c *AppValue) Boot(ctx context.Context) {
	// Watch for configmap changes.
	go c.watchConfigMap(ctx)

	// Watch for secret changes.
	go c.watchSecret(ctx)

	// Build a cache of configmaps and link each app to its configmaps.
	go c.buildCache(ctx)
}
