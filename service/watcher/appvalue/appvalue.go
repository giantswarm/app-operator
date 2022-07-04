package appvalue

import (
	"context"
	"sync"

	"github.com/giantswarm/k8sclient/v6/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/app-operator/v7/pkg/label"
	"github.com/giantswarm/app-operator/v7/service/internal/recorder"
)

type AppValueWatcherConfig struct {
	Event     recorder.Interface
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	// SecretNamespace is used to limit access to secrets to only the
	// SecretNamespace. No other namespaces will be watched for Secrets.
	SecretNamespace   string
	UniqueApp         bool
	WorkloadClusterID string
}

type AppValueWatcher struct {
	event     recorder.Interface
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	resourcesToApps sync.Map
	secretNamespace string
	selector        labels.Selector
}

func NewAppValueWatcher(config AppValueWatcherConfig) (*AppValueWatcher, error) {
	if config.Event == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Event must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.SecretNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.SecretNamespace must not be empty", config)
	}

	var selector labels.Selector
	{
		if config.WorkloadClusterID != "" {
			selector = label.ClusterSelector(config.WorkloadClusterID)
		} else {
			selector = label.AppVersionSelector(config.UniqueApp)
		}
	}

	c := &AppValueWatcher{
		event:     config.Event,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		resourcesToApps: sync.Map{},
		secretNamespace: config.SecretNamespace,
		selector:        selector,
	}

	return c, nil
}

func (c *AppValueWatcher) Boot(ctx context.Context) {
	// Watch for configmap changes.
	go c.watchConfigMap(ctx)

	// Watch for secret changes.
	go c.watchSecret(ctx)

	// Build a cache of configmaps and link each app to its configmaps.
	go c.buildCache(ctx)
}
