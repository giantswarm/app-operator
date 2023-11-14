package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/giantswarm/app-operator/v6/flag"
	"github.com/giantswarm/app-operator/v6/pkg/env"
	"github.com/giantswarm/app-operator/v6/pkg/project"
	"github.com/giantswarm/app-operator/v6/service/controller/app"
	"github.com/giantswarm/app-operator/v6/service/controller/catalog"
	"github.com/giantswarm/app-operator/v6/service/internal/clientcache"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache"
	"github.com/giantswarm/app-operator/v6/service/internal/recorder"
	"github.com/giantswarm/app-operator/v6/service/watcher/appvalue"
	"github.com/giantswarm/app-operator/v6/service/watcher/chartstatus"
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
	appController      *app.App
	catalogController  *catalog.Catalog
	appValueWatcher    *appvalue.AppValueWatcher
	chartStatusWatcher *chartstatus.ChartStatusWatcher
	bootOnce           sync.Once

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

	var catalogController *catalog.Catalog
	{
		c := catalog.Config{
			Logger:    config.Logger,
			K8sClient: config.K8sClient,

			MaxEntriesPerApp: config.Viper.GetInt(config.Flag.Service.AppCatalog.MaxEntriesPerApp),
			Provider:         config.Viper.GetString(config.Flag.Service.Provider.Kind),
			UniqueApp:        config.Viper.GetBool(config.Flag.Service.App.Unique),
		}

		catalogController, err = catalog.NewCatalog(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	fs := afero.NewOsFs()
	podNamespace := env.PodNamespace()

	fmt.Printf("\nDisable cache: %t\n\n", config.Viper.GetBool(config.Flag.Service.Kubernetes.DisableClientCache))
	var clientCache *clientcache.Resource
	{
		c := clientcache.Config{
			Fs:        fs,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			HTTPClientTimeout: config.Viper.GetDuration(config.Flag.Service.Helm.HTTP.ClientTimeout),
			DisableCache:      config.Viper.GetBool(config.Flag.Service.Kubernetes.DisableClientCache),
		}

		clientCache, err = clientcache.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var indexCache indexcache.Interface
	{
		c := indexcache.Config{
			Logger: config.Logger,

			HTTPClientTimeout: config.Viper.GetDuration(config.Flag.Service.Helm.HTTP.ClientTimeout),
		}

		indexCache, err = indexcache.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appController *app.App
	{
		c := app.Config{
			ClientCache: clientCache,
			Fs:          fs,
			IndexCache:  indexCache,
			Logger:      config.Logger,
			K8sClient:   config.K8sClient,

			ChartNamespace:               config.Viper.GetString(config.Flag.Service.Chart.Namespace),
			HTTPClientTimeout:            config.Viper.GetDuration(config.Flag.Service.Helm.HTTP.ClientTimeout),
			ImageRegistry:                config.Viper.GetString(config.Flag.Service.Image.Registry),
			PodNamespace:                 podNamespace,
			Provider:                     config.Viper.GetString(config.Flag.Service.Provider.Kind),
			ResyncPeriod:                 config.Viper.GetDuration(config.Flag.Service.Operatorkit.ResyncPeriod),
			UniqueApp:                    config.Viper.GetBool(config.Flag.Service.App.Unique),
			WatchNamespace:               config.Viper.GetString(config.Flag.Service.App.WatchNamespace),
			WorkloadClusterID:            config.Viper.GetString(config.Flag.Service.App.WorkloadClusterID),
			DependencyWaitTimeoutMinutes: config.Viper.GetInt(config.Flag.Service.App.DependencyWaitTimeoutMinutes),
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

			SecretNamespace:   podNamespace,
			UniqueApp:         config.Viper.GetBool(config.Flag.Service.App.Unique),
			WorkloadClusterID: config.Viper.GetString(config.Flag.Service.App.WorkloadClusterID),
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

			ChartNamespace:    config.Viper.GetString(config.Flag.Service.Chart.Namespace),
			PodNamespace:      podNamespace,
			UniqueApp:         config.Viper.GetBool(config.Flag.Service.App.Unique),
			WorkloadClusterID: config.Viper.GetString(config.Flag.Service.App.WorkloadClusterID),
		}

		chartStatusWatcher, err = chartstatus.NewChartStatusWatcher(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var versionService *version.Service
	{
		versionConfig := version.Config{
			Description: project.Description(),
			GitCommit:   project.GitSHA(),
			Name:        project.Name(),
			Source:      project.Source(),
			Version:     project.Version(),
		}

		versionService, err = version.New(versionConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	newService := &Service{
		Version: versionService,

		appController:      appController,
		catalogController:  catalogController,
		appValueWatcher:    appValueWatcher,
		chartStatusWatcher: chartStatusWatcher,
		bootOnce:           sync.Once{},

		unique: config.Viper.GetBool(config.Flag.Service.App.Unique),
	}

	return newService, nil
}

// Boot starts top level service implementation.
func (s *Service) Boot(ctx context.Context) {
	s.bootOnce.Do(func() {
		// Boot appCatalogController only if it's unique app.
		if s.unique {
			go s.catalogController.Boot(ctx)
		}

		// Start the controller.
		go s.appController.Boot(ctx)

		// Start the watchers.
		go s.appValueWatcher.Boot(ctx)
		go s.chartStatusWatcher.Boot(ctx)
	})
}
