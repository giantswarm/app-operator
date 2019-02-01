// +build k8srequired

package chart

import (
	"context"
	"testing"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"

	"github.com/giantswarm/app-operator/integration/setup"
)

var (
	h          *framework.Host
	helmClient *helmclient.Client
	l          micrologger.Logger
	crdClient  *k8scrdclient.CRDClient
)

func init() {
	var err error

	{
		c := micrologger.Config{}
		l, err = micrologger.New(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := framework.HostConfig{
			Logger: l,

			ClusterID:       "n/a",
			VaultToken:      "n/a",
			TargetNamespace: "giantswarm",
		}

		h, err = framework.NewHost(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := helmclient.Config{
			Logger:          l,
			K8sClient:       h.K8sClient(),
			RestConfig:      h.RestConfig(),
			TillerNamespace: "giantswarm",
		}
		helmClient, err = helmclient.New(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := k8scrdclient.Config{
			K8sExtClient: h.ExtClient(),
			Logger:       l,
		}

		crdClient, err = k8scrdclient.New(c)
		if err != nil {
			panic(err.Error())
		}
	}
}

// TestMain allows us to have common setup and teardown steps that are run
// once for all the tests https://golang.org/pkg/testing/#hdr-Main.
func TestMain(m *testing.M) {
	ctx := context.Background()
	setup.WrapTestMain(ctx, h, crdClient, helmClient, l, m)
}
