package service

import (
	"context"
	"sync"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/versionbundle"
	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/giantswarm/app-operator/v3/flag"
	"github.com/giantswarm/app-operator/v3/pkg/env"
	"github.com/giantswarm/app-operator/v3/pkg/project"
	"github.com/giantswarm/app-operator/v3/service/controller/app"
	"github.com/giantswarm/app-operator/v3/service/controller/appcatalog"
	"github.com/giantswarm/app-operator/v3/service/watcher/appvalue"
	"github.com/giantswarm/app-operator/v3/service/watcher/chartstatus"
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

	var appController *app.App
	{
		c := app.Config{
			Fs:        fs,
			Logger:    config.Logger,
			K8sClient: config.K8sClient,

			ChartNamespace:    config.Viper.GetString(config.Flag.Service.Chart.Namespace),
			HTTPClientTimeout: config.Viper.GetDuration(config.Flag.Service.Helm.HTTP.ClientTimeout),
			ImageRegistry:     config.Viper.GetString(config.Flag.Service.Image.Registry),
			PodNamespace:      podNamespace,
			ResyncPeriod:      config.Viper.GetDuration(config.Flag.Service.Operatorkit.ResyncPeriod),
			UniqueApp:         config.Viper.GetBool(config.Flag.Service.App.Unique),
			WebhookAuthToken:  config.Viper.GetString(config.Flag.Service.Chart.WebhookAuthToken),
			WebhookBaseURL:    config.Viper.GetString(config.Flag.Service.Chart.WebhookBaseURL),
		}

		appController, err = app.NewApp(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appValueWatcher *appvalue.AppValueWatcher
	{
		c := appvalue.AppValueWatcherConfig{
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
