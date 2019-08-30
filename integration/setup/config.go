// +build k8srequired

package setup

import (
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2esetup/chart/env"
	"github.com/giantswarm/e2esetup/k8s"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	namespace = "giantswarm"
)

type Config struct {
	Guest      *framework.Guest
	HelmClient *helmclient.Client
	K8s        *k8s.Setup
	K8sClients *k8sclient.Clients
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

	var guest *framework.Guest
	{
		c := framework.GuestConfig{
			Logger: logger,

			ClusterID:    "n/a",
			CommonDomain: "n/a",
		}

		guest, err = framework.NewGuest(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

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

	var k8sSetup *k8s.Setup
	{
		c := k8s.SetupConfig{
			Clients: cpK8sClients,
			Logger:  logger,
		}

		k8sSetup, err = k8s.NewSetup(c)
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

	var helmClient *helmclient.Client
	{
		c := helmclient.Config{
			Logger:    logger,
			K8sClient: cpK8sClients.K8sClient(),

			RestConfig:      cpK8sClients.RestConfig(),
			TillerNamespace: namespace,
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var newRelease *release.Release
	{
		c := release.Config{
			ExtClient:  cpK8sClients.ExtClient(),
			G8sClient:  cpK8sClients.G8sClient(),
			HelmClient: helmClient,
			K8sClient:  cpK8sClients.K8sClient(),
			Logger:     logger,

			Namespace: namespace,
		}

		newRelease, err = release.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	c := Config{
		Guest:      guest,
		HelmClient: helmClient,
		K8s:        k8sSetup,
		K8sClients: cpK8sClients,
		KubeConfig: kubeConfig,
		Logger:     logger,
		Release:    newRelease,
	}

	return c, nil
}
