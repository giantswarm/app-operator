// +build k8srequired

package setup

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/pkg/crd"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/app-operator/integration/key"
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

	// Create AppCatalog and App CRDs. The Chart CRD is created by the operator
	// for the kubeconfig test that bootstraps chart-operator.
	crds := []string{
		"AppCatalog",
		"App",
	}

	{
		for _, crdName := range crds {
			config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring %#q CRD exists", crdName))

			err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("application.giantswarm.io", crdName), backoff.NewMaxRetries(7, 1*time.Second))
			if err != nil {
				return microerror.Mask(err)
			}

			config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured %#q CRD exists", crdName))
		}
	}

	var operatorTarballPath string
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "getting tarball URL")

		operatorTarballURL, err := appcatalog.GetLatestChart(ctx, key.ControlPlaneTestCatalogStorageURL(), project.Name(), project.Version())
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball URL is %#q", operatorTarballURL))

		config.Logger.LogCtx(ctx, "level", "debug", "message", "pulling tarball")

		operatorTarballPath, err = config.HelmClient.PullChartTarball(ctx, operatorTarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path is %#q", operatorTarballPath))
	}

	{
		defer func() {
			fs := afero.NewOsFs()
			err := fs.Remove(operatorTarballPath)
			if err != nil {
				config.Logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", operatorTarballPath), "stack", fmt.Sprintf("%#v", err))
			}
		}()

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing %#q", project.Name()))

		appOperatorValues := map[string]interface{}{
			"Installation": map[string]interface{}{
				"V1": map[string]interface{}{
					"Registry": map[string]interface{}{
						"Domain": "quay.io",
					},
				},
			},
		}
		opts := helmclient.InstallOptions{
			ReleaseName: project.Name(),
		}

		err = config.HelmClient.InstallReleaseFromTarball(ctx,
			operatorTarballPath,
			namespace,
			appOperatorValues,
			opts)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed %#q", project.Name()))
	}

	return nil
}
