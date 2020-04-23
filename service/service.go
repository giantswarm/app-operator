package service

import (
	"context"
	"sync"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/k8sclient/k8srestconfig"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/versionbundle"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/app-operator/flag"
	"github.com/giantswarm/app-operator/pkg/project"
	"github.com/giantswarm/app-operator/service/collector"
	"github.com/giantswarm/app-operator/service/controller/app"
	"github.com/giantswarm/app-operator/service/controller/appcatalog"
)

// Config represents the configuration used to create a new service.
type Config struct {
	Logger micrologger.Logger
	Flag   *flag.Flag

	Viper *viper.Viper
}

// Service is a type providing implementation of microkit service interface.
type Service struct {
	Version *version.Service

	// Internals
	appController        *app.App
	appCatalogController *appcatalog.AppCatalog
	bootOnce             sync.Once
	operatorCollector    *collector.Set
}

// New creates a new service with given configuration.
func New(config Config) (*Service, error) {
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Flag must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Viper must not be empty", config)
	}

	var err error

	var restConfig *rest.Config
	{
		c := k8srestconfig.Config{
			Logger: config.Logger,

			Address:    config.Viper.GetString(config.Flag.Service.Kubernetes.Address),
			InCluster:  config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster),
			KubeConfig: config.Viper.GetString(config.Flag.Service.Kubernetes.KubeConfig),
			TLS: k8srestconfig.ConfigTLS{
				CAFile:  config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile),
				CrtFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile),
				KeyFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile),
			},
		}

		restConfig, err = k8srestconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var k8sClient k8sclient.Interface
	{
		c := k8sclient.ClientsConfig{
			Logger: config.Logger,
			SchemeBuilder: k8sclient.SchemeBuilder{
				applicationv1alpha1.AddToScheme,
			},

			RestConfig: restConfig,
		}

		k8sClient, err = k8sclient.NewClients(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appCatalogController *appcatalog.AppCatalog
	{
		c := appcatalog.Config{
			Logger:    config.Logger,
			K8sClient: k8sClient,
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
			K8sClient: k8sClient,

			ChartNamespace: config.Viper.GetString(config.Flag.Service.Chart.Namespace),
			ImageRegistry:  config.Viper.GetString(config.Flag.Service.Image.Registry),
		}

		appController, err = app.NewApp(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorCollector *collector.Set
	{
		c := collector.SetConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,
		}

		operatorCollector, err = collector.NewSet(c)
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
		bootOnce:             sync.Once{},
		operatorCollector:    operatorCollector,
	}

	return newService, nil
}

// Boot starts top level service implementation.
func (s *Service) Boot(ctx context.Context) {
	s.bootOnce.Do(func() {
		go s.operatorCollector.Boot(ctx)

		// Start the controllers.
		go s.appCatalogController.Boot(ctx)
		go s.appController.Boot(ctx)
	})
}
