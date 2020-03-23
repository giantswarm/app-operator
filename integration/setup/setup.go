// +build k8srequired

package setup

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"k8s.io/helm/pkg/helm"

	"github.com/giantswarm/app-operator/integration/env"
	"github.com/giantswarm/app-operator/integration/key"
	"github.com/giantswarm/app-operator/integration/templates"
	"github.com/giantswarm/app-operator/pkg/project"
)

func Setup(m *testing.M, config Config) {
	ctx := context.Background()

	var v int
	var err error

	err = installResources(ctx, config)
	if err != nil {
		config.Logger.LogCtx(ctx, "level", "error", "message", "failed to install app-operator dependent resources", "stack", fmt.Sprintf("%#v", err))
		v = 1
	}

	if v == 0 {
		if err != nil {
			config.Logger.LogCtx(ctx, "level", "error", "message", "failed to create operator resources", "stack", fmt.Sprintf("%#v", err))
			v = 1
		}
	}

	if v == 0 {
		v = m.Run()
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
		err = config.HelmClient.EnsureTillerInstalled(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing app-operator"))

		operatorVersion := fmt.Sprintf("1.0.0-%s", env.CircleSHA())
		operatorTarballPath, err := config.ApprClient.PullChartTarballFromRelease(ctx, key.AppOperatorChartName(), operatorVersion)
		if err != nil {
			return microerror.Mask(err)
		}

		defer func() {
			fs := afero.NewOsFs()
			err := fs.Remove(operatorTarballPath)
			if err != nil {
				config.Logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", operatorTarballPath), "stack", fmt.Sprintf("%#v", err))
			}
		}()

		err = config.HelmClient.InstallReleaseFromTarball(ctx,
			operatorTarballPath,
			namespace,
			helm.ReleaseName(project.Name()),
			helm.ValueOverrides([]byte(templates.AppOperatorValues)))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed app-operator"))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for app-operator pod"))

		err = config.Release.PodExists(ctx, namespace, "app=app-operator")
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for app-operator pod"))
	}

	return nil
}
