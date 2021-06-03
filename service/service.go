package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/versionbundle"
	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/giantswarm/app-operator/v4/flag"
	"github.com/giantswarm/app-operator/v4/pkg/env"
	"github.com/giantswarm/app-operator/v4/pkg/project"
	"github.com/giantswarm/app-operator/v4/service/controller/app"
	"github.com/giantswarm/app-operator/v4/service/controller/appcatalog"
	"github.com/giantswarm/app-operator/v4/service/internal/clientcache"
	"github.com/giantswarm/app-operator/v4/service/internal/crdcache"
	"github.com/giantswarm/app-operator/v4/service/internal/recorder"
	"github.com/giantswarm/app-operator/v4/service/watcher/appvalue"
	"github.com/giantswarm/app-operator/v4/service/watcher/chartstatus"
)

// Config represents the configuration used to create a new service.
type Config struct {
	Logger    micrologger.Logger
	K8sClient k8sclient.Interface

	Flag  *flag.Flag
	Viper *viper.Viper
}

// Service is a type providing implementation of microkit service interface.
type Service struct {
	Version *version.Service

	// Internals
	appController        *app.App
	appCatalogController *appcatalog.AppCatalog
	appValueWatcher      *appvalue.AppValueWatcher
	chartStatusWatcher   *chartstatus.ChartStatusWatcher
	bootOnce             sync.Once

	// Settings
	unique bool
}

// New creates a new service with given configuration.
func New(config Config) (*Service, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Flag must not be empty", config)
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Viper must not be empty", config)
	}

	var err error

	var appCatalogController *appcatalog.AppCatalog
	{
		c := appcatalog.Config{
			Logger:    config.Logger,
			K8sClient: config.K8sClient,

			MaxEntriesPerApp: config.Viper.GetInt(config.Flag.Service.AppCatalog.MaxEntriesPerApp),
			UniqueApp:        config.Viper.GetBool(config.Flag.Service.App.Unique),
		}

		appCatalogController, err = appcatalog.NewAppCatalog(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	fs := afero.NewOsFs()
	podNamespace := env.PodNamespace()

	var clientCache *clientcache.Resource
	{
		c := clientcache.Config{
			Fs:        fs,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			HTTPClientTimeout: config.Viper.GetDuration(config.Flag.Service.Helm.HTTP.ClientTimeout),
		}

		clientCache, err = clientcache.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var crdCache *crdcache.Resource
	{
		c := crdcache.Config{
			Logger: config.Logger,
		}

		crdCache, err = crdcache.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appController *app.App
	{
		c := app.Config{
			ClientCache: clientCache,
			CRDCache:    crdCache,
			Fs:          fs,
			Logger:      config.Logger,
			K8sClient:   config.K8sClient,

			ChartNamespace:    config.Viper.GetString(config.Flag.Service.Chart.Namespace),
			HTTPClientTimeout: config.Viper.GetDuration(config.Flag.Service.Helm.HTTP.ClientTimeout),
			ImageRegistry:     config.Viper.GetString(config.Flag.Service.Image.Registry),
			PodNamespace:      podNamespace,
			Provider:          config.Viper.GetString(config.Flag.Service.Provider.Kind),
			ResyncPeriod:      config.Viper.GetDuration(config.Flag.Service.Operatorkit.ResyncPeriod),
			UniqueApp:         config.Viper.GetBool(config.Flag.Service.App.Unique),
		}

		appController, err = app.NewApp(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var event recorder.Interface
	{
		c := recorder.Config{
			K8sClient: config.K8sClient,

			Component: fmt.Sprintf("%s-%s", project.Name(), project.Version()),
		}

		event = recorder.New(c)
	}

	var appValueWatcher *appvalue.AppValueWatcher
	{
		c := appvalue.AppValueWatcherConfig{
			Event:     event,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			UniqueApp: config.Viper.GetBool(config.Flag.Service.App.Unique),
		}

		appValueWatcher, err = appvalue.NewAppValueWatcher(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var chartStatusWatcher *chartstatus.ChartStatusWatcher
	{
		c := chartstatus.ChartStatusWatcherConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			ChartNamespace: config.Viper.GetString(config.Flag.Service.Chart.Namespace),
			PodNamespace:   podNamespace,
			UniqueApp:      config.Viper.GetBool(config.Flag.Service.App.Unique),
		}

		chartStatusWatcher, err = chartstatus.NewChartStatusWatcher(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var versionService *version.Service
	{
		versionConfig := version.Config{
			Description:    project.Description(),
			GitCommit:      project.GitSHA(),
			Name:           project.Name(),
			Source:         project.Source(),
			Version:        project.Version(),
			VersionBundles: []versionbundle.Bundle{project.NewVersionBundle()},
		}

		versionService, err = version.New(versionConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	newService := &Service{
		Version: versionService,

		appController:        appController,
		appCatalogController: appCatalogController,
		appValueWatcher:      appValueWatcher,
		chartStatusWatcher:   chartStatusWatcher,
		bootOnce:             sync.Once{},

		unique: config.Viper.GetBool(config.Flag.Service.App.Unique),
	}

	return newService, nil
}

// Boot starts top level service implementation.
func (s *Service) Boot(ctx context.Context) {
	s.bootOnce.Do(func() {
		// Boot appCatalogController only if it's unique app.
		if s.unique {
			go s.appCatalogController.Boot(ctx)
		}

		// Start the controller.
		go s.appController.Boot(ctx)

		// Start the watchers.
		go s.appValueWatcher.Boot(ctx)
		go s.chartStatusWatcher.Boot(ctx)
	})
}
