// +build k8srequired

package setup

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"

	"github.com/giantswarm/app-operator/integration/env"
	"github.com/giantswarm/app-operator/integration/teardown"
)

func WrapTestMain(ctx context.Context, h *framework.Host, crdClient *k8scrdclient.CRDClient, helmClient *helmclient.Client, l micrologger.Logger, m *testing.M) {
	var v int
	var err error

	err = h.CreateNamespace("giantswarm")
	if err != nil {
		log.Printf("%#v\n", err)
		v = 1
	}

	err = helmClient.EnsureTillerInstalled(ctx)
	if err != nil {
		log.Printf("%#v\n", err)
		v = 1
	}

	err = resources(ctx, h, crdClient, l)
	if err != nil {
		log.Printf("%#v\n", err)
		v = 1
	}

	if v == 0 {
		v = m.Run()
	}

	if env.KeepResources() != "true" {
		// only do full teardown when not on CI
		if env.CircleCI() != "true" {
			err := teardown.Teardown(h, helmClient)
			if err != nil {
				log.Printf("%#v\n", err)
				v = 1
			}
			// TODO there should be error handling for the framework teardown.
			h.Teardown()
		}
	}

	os.Exit(v)
}

func resources(ctx context.Context, h *framework.Host, crdClient *k8scrdclient.CRDClient, l micrologger.Logger) error {
	version := fmt.Sprintf(":%s", env.CircleSHA())

	var err error

	err = crdClient.EnsureCreated(ctx, v1alpha1.NewChartCRD(), backoff.NewExponential(backoff.ShortMaxWait, backoff.ShortMaxInterval))
	if err != nil {
		return microerror.Mask(err)
	}

	err = h.InstallStableOperator("app-operator", "app", "")
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
