//go:build k8srequired
// +build k8srequired

package setup

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/apptest"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"

	prometheusMonitoringV1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/giantswarm/app-operator/v6/integration/env"
	"github.com/giantswarm/app-operator/v6/integration/release"
)

type Config struct {
	AppTest               apptest.Interface
	HelmClient            helmclient.Interface
	HelmControllerBackend bool
	K8s                   *k8sclient.Setup
	K8sClients            k8sclient.Interface
	KubeConfig            *kubeconfig.KubeConfig
	Release               *release.Release
	Logger                micrologger.Logger
}

func NewConfig() (Config, error) {
	var err error

	var logger micrologger.Logger
	{
		c := micrologger.Config{}

		logger, err = micrologger.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	fs := afero.NewOsFs()

	var appTest apptest.Interface
	{
		c := apptest.Config{
			Logger: logger,

			KubeConfigPath: env.KubeConfigPath(),
		}

		appTest, err = apptest.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var cpK8sClients *k8sclient.Clients
	{
		c := k8sclient.ClientsConfig{
			Logger: logger,
			SchemeBuilder: k8sclient.SchemeBuilder{
				prometheusMonitoringV1.AddToScheme,
				v1alpha1.AddToScheme,
				sourcev1beta2.AddToScheme,
				helmv2.AddToScheme,
			},

			KubeConfigPath: env.KubeConfigPath(),
		}

		cpK8sClients, err = k8sclient.NewClients(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var k8sSetup *k8sclient.Setup
	{
		c := k8sclient.SetupConfig{
			Clients: cpK8sClients,
			Logger:  logger,
		}

		k8sSetup, err = k8sclient.NewSetup(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var kubeConfig *kubeconfig.KubeConfig
	{
		c := kubeconfig.Config{
			Logger:    logger,
			K8sClient: cpK8sClients.K8sClient(),
		}

		kubeConfig, err = kubeconfig.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var helmClient helmclient.Interface
	{
		c := helmclient.Config{
			Fs:         fs,
			K8sClient:  cpK8sClients.K8sClient(),
			Logger:     logger,
			RestClient: cpK8sClients.RESTClient(),
			RestConfig: cpK8sClients.RESTConfig(),
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var newRelease *release.Release
	{
		c := release.Config{
			HelmClient: helmClient,
			K8sClient:  cpK8sClients,
			Logger:     logger,
		}

		newRelease, err = release.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	c := Config{
		AppTest:    appTest,
		HelmClient: helmClient,
		K8s:        k8sSetup,
		K8sClients: cpK8sClients,
		KubeConfig: kubeConfig,
		Logger:     logger,
		Release:    newRelease,
	}

	return c, nil
}
