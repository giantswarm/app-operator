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

	"github.com/giantswarm/app-operator/v2/flag"
	"github.com/giantswarm/app-operator/v2/pkg/project"
	"github.com/giantswarm/app-operator/v2/service/controller/app"
	"github.com/giantswarm/app-operator/v2/service/controller/appcatalog"
	"github.com/giantswarm/app-operator/v2/service/watcher"
)

// Config represents the configuration used to create a new service.
type Config struct {
	Logger micrologger.Logger
	Flag   *flag.Flag

	Viper     *viper.Viper
	K8sClient k8sclient.Interface
}

// Service is a type providing implementation of microkit service interface.
type Service struct {
	Version *version.Service

	// Internals
	appController        *app.App
	appCatalogController *appcatalog.AppCatalog
	bootOnce             sync.Once
	configMapWatcher     *watcher.AppValueWatcher
}

// New creates a new service with given configuration.
func New(config Config) (*Service, error) {
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Flag must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
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

			UniqueApp: config.Viper.GetBool(config.Flag.Service.App.Unique),
		}

		appCatalogController, err = appcatalog.NewAppCatalog(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	fs := afero.NewOsFs()
	var appController *app.App
	{
		c := app.Config{
			Fs:        fs,
			Logger:    config.Logger,
			K8sClient: config.K8sClient,

			ChartNamespace:    config.Viper.GetString(config.Flag.Service.Chart.Namespace),
			HTTPClientTimeout: config.Viper.GetDuration(config.Flag.Service.Helm.HTTP.ClientTimeout),
			ImageRegistry:     config.Viper.GetString(config.Flag.Service.Image.Registry),
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

	var appValueWatcher *watcher.AppValueWatcher
	{
		c := watcher.AppValueWatcherConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			UniqueApp: config.Viper.GetBool(config.Flag.Service.App.Unique),
		}

		appValueWatcher, err = watcher.NewAppValueWatcher(c)
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
		configMapWatcher:     appValueWatcher,
		bootOnce:             sync.Once{},
	}

	return newService, nil
}

// Boot starts top level service implementation.
func (s *Service) Boot(ctx context.Context) {
	s.bootOnce.Do(func() {
		// Start the controllers.
		go s.appCatalogController.Boot(ctx)
		go s.appController.Boot(ctx)
		go s.configMapWatcher.Boot(ctx)
	})
}
