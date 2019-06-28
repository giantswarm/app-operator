// +build k8srequired

package setup

import (
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/e2e-harness/pkg/harness"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2esetup/k8s"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	namespace       = "giantswarm"
	tillerNamespace = "kube-system"
)

type Config struct {
	Guest      *framework.Guest
	HelmClient *helmclient.Client
	Host       *framework.Host
	K8s        *k8s.Setup
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

	var host *framework.Host
	{
		c := framework.HostConfig{
			Logger: logger,

			ClusterID: "n/a",
		}

		host, err = framework.NewHost(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var cpK8sClients *k8s.Clients
	{
		c := k8s.ClientsConfig{
			Logger: logger,

			KubeConfigPath: harness.DefaultKubeConfig,
		}

		cpK8sClients, err = k8s.NewClients(c)
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
			K8sClient: host.K8sClient(),
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
			K8sClient: host.K8sClient(),

			RestConfig:      host.RestConfig(),
			TillerNamespace: tillerNamespace,
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var newRelease *release.Release
	{
		c := release.Config{
			ExtClient:  host.ExtClient(),
			G8sClient:  host.G8sClient(),
			HelmClient: helmClient,
			K8sClient:  host.K8sClient(),
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
		Host:       host,
		K8s:        k8sSetup,
		KubeConfig: kubeConfig,
		Logger:     logger,
		Release:    newRelease,
	}

	return c, nil
}
