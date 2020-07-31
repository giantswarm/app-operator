package collector

import (
	"github.com/giantswarm/exporterkit/collector"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type SetConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	AppTeamMapping map[string]string
	DefaultTeam    string
	UniqueApp      bool
}

// Set is basically only a wrapper for the operator's collector implementations.
// It eases the iniitialization and prevents some weird import mess so we do not
// have to alias packages.
type Set struct {
	*collector.Set
}

func NewSet(config SetConfig) (*Set, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

	var appCollector *App
	{
		c := AppConfig{ // nolint:gosimple
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			AppTeamMapping: config.AppTeamMapping,
			DefaultTeam:    config.DefaultTeam,
			UniqueApp:      config.UniqueApp,
		}

		appCollector, err = NewApp(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appOperatorCollector *AppOperator
	{
		c := AppOperatorConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		appOperatorCollector, err = NewAppOperator(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var collectorSet *collector.Set
	{
		c := collector.SetConfig{
			Collectors: []collector.Interface{
				appCollector,
			},
			Logger: config.Logger,
		}

		if config.UniqueApp {
			config.Logger.Log("level", "debug", "message", "app-operator collector is enabled")
			c.Collectors = append(c.Collectors, appOperatorCollector)
		}

		collectorSet, err = collector.NewSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Set{
		Set: collectorSet,
	}

	return s, nil
}
