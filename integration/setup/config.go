// +build k8srequired

package setup

import (
	"github.com/giantswarm/e2esetup/chart/env"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"

	"github.com/giantswarm/app-operator/integration/release"
)

const (
	namespace = "giantswarm"
)

type Config struct {
	HelmClient helmclient.Interface
	K8s        *k8sclient.Setup
	K8sClients k8sclient.Interface
	KubeConfig *kubeconfig.KubeConfig
	Release    *release.Release
	Logger     micrologger.Logger
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

	var cpK8sClients *k8sclient.Clients
	{
		c := k8sclient.ClientsConfig{
			Logger: logger,

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
			Fs:        fs,
			Logger:    logger,
			K8sClient: cpK8sClients,
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
		HelmClient: helmClient,
		K8s:        k8sSetup,
		K8sClients: cpK8sClients,
		KubeConfig: kubeConfig,
		Logger:     logger,
		Release:    newRelease,
	}

	return c, nil
}
