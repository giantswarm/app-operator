// +build k8srequired

package setup

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/integration/env"
	"github.com/giantswarm/app-operator/integration/key"
)

func Setup(m *testing.M, config Config) {
	ctx := context.Background()

	var v int
	var err error

	err = installResources(ctx, config)
	if err != nil {
		config.Logger.LogCtx(ctx, "level", "error", "message", "failed to install resources", "stack", fmt.Sprintf("%#v", err))
		v = 1
	}

	if v == 0 && config.UseDefaultTenant {
		if err != nil {
			config.Logger.LogCtx(ctx, "level", "error", "message", "failed to create tenant cluster", "stack", fmt.Sprintf("%#v", err))
			v = 1
		}
	}

	if v == 0 {
		v = m.Run()
	}

	if !env.KeepResources() {
		if config.UseDefaultTenant {
			if err != nil {
				config.Logger.LogCtx(ctx, "level", "error", "message", "failed to delete tenant cluster", "stack", fmt.Sprintf("%#v", err))
				v = 1
			}
		}

		if !env.CircleCI() {
			err := teardown(ctx, config)
			if err != nil {
				// teardown errors are logged inside the function.
				v = 1
			}
		}
	}

	os.Exit(v)
}

func installResources(ctx context.Context, config Config) error {
	var err error

	{
		err = config.K8s.EnsureNamespaceCreated(ctx, namespace)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = config.Release.InstallOperator(ctx, key.AppOperatorReleaseName(), release.NewVersion(env.CircleSHA()), "", v1alpha1.NewAppCRD())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
