//go:build k8srequired
// +build k8srequired

package basic

import (
	"context"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/apptest"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/spf13/afero"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/app-operator/v5/integration/key"
	"github.com/giantswarm/app-operator/v5/integration/templates"
)

// TestAppLifecycle tests a chart CR can be created, updated and deleted
// using a app, appCatalog CRs processed by app-operator.
//
// - Install chart-operator.
// - Create test app CR.
// - Ensure chart CR specified in the app CR is deployed.
//
// - Update app version in app CR.
// - Ensure chart CR is redeployed using updated app CR information.
//
// - Delete app CR
// - Ensure chart CR is deleted.
//
func TestAppLifecycle(t *testing.T) {
	ctx := context.Background()

	var chart v1alpha1.Chart
	var cr v1alpha1.App
	var err error

	{
		{
			crdName := "Chart"
			config.Logger.Debugf(ctx, "ensuring %#q CRD exists", crdName)

			crd, err := config.CRDGetter.LoadCRD(ctx, "application.giantswarm.io", crdName)
			if err != nil {
				t.Fatalf("expected %#v got %#v", nil, err)
			}

			err = config.K8sClients.CtrlClient().Create(ctx, crd)
			if apierrors.IsAlreadyExists(err) {
				// no-op
			} else if err != nil {
				t.Fatalf("expected %#v got %#v", nil, err)
			}

			config.Logger.Debugf(ctx, "ensured %#q CRD exists", crdName)
		}

		var tarballPath string
		{
			config.Logger.Debugf(ctx, "installing %#q", key.ChartOperatorName())

			tarballURL, err := appcatalog.GetLatestChart(ctx, key.DefaultCatalogStorageURL(), key.ChartOperatorName(), key.ChartOperatorVersion())
			if err != nil {
				t.Fatalf("expected %#v got %#v", nil, err)
			}

			tarballPath, err = config.HelmClient.PullChartTarball(ctx, tarballURL)
			if err != nil {
				t.Fatalf("expected %#v got %#v", nil, err)
			}

			defer func() {
				fs := afero.NewOsFs()
				err := fs.Remove(tarballPath)
				if err != nil {
					t.Fatalf("expected %#v got %#v", nil, err)
				}
			}()
		}

		var values map[string]interface{}
		{
			err = yaml.Unmarshal([]byte(templates.ChartOperatorValues), &values)
			if err != nil {
				t.Fatalf("expected %#v got %#v", nil, err)
			}
		}

		opts := helmclient.InstallOptions{
			ReleaseName: key.ChartOperatorName(),
			Wait:        true,
		}
		err = config.HelmClient.InstallReleaseFromTarball(ctx, tarballPath, key.GiantSwarmNamespace(), values, opts)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "installed %#q", key.ChartOperatorName())
	}

	{
		apps := []apptest.App{
			{
				// chart-operator app CR is used by the chart status watcher
				// to get a kubeconfig.
				AppCRName:     key.ChartOperatorName(),
				CatalogName:   key.DefaultCatalogName(),
				Name:          key.ChartOperatorName(),
				Namespace:     key.GiantSwarmNamespace(),
				ValuesYAML:    templates.ChartOperatorValues,
				Version:       key.ChartOperatorVersion(),
				WaitForDeploy: false,
			},
			{
				// Install test app.
				CatalogName:   key.DefaultCatalogName(),
				Name:          key.TestAppName(),
				Namespace:     key.GiantSwarmNamespace(),
				Version:       "0.1.0",
				WaitForDeploy: true,
			},
		}
		err = config.AppTest.InstallApps(ctx, apps)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	{
		config.Logger.Debugf(ctx, "checking tarball URL in chart spec")

		tarballURL := "https://giantswarm.github.io/default-catalog/test-app-0.1.0.tgz"
		err = config.K8sClients.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: key.TestAppName(), Namespace: key.GiantSwarmNamespace()},
			&chart,
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		if chart.Spec.TarballURL != tarballURL {
			t.Fatalf("expected tarballURL: %#q got %#q", tarballURL, chart.Spec.TarballURL)
		}
		if chart.Labels[label.ChartOperatorVersion] != "1.0.0" {
			t.Fatalf("expected version label: %#q got %#q", "1.0.0", chart.Labels[label.ChartOperatorVersion])
		}

		config.Logger.Debugf(ctx, "checked tarball URL in chart spec")
	}

	{
		config.Logger.Debugf(ctx, "updating app %#q", key.TestAppName())

		err = config.K8sClients.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: key.TestAppName(), Namespace: key.GiantSwarmNamespace()},
			&cr,
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		cr.Spec.Version = "0.1.1"
		err = config.K8sClients.CtrlClient().Update(ctx, &cr)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "updated app %#q", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "checking tarball URL in chart spec")

		err = config.Release.WaitForReleaseVersion(ctx, key.GiantSwarmNamespace(), key.TestAppName(), "0.1.1")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.K8sClients.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: key.TestAppName(), Namespace: key.GiantSwarmNamespace()},
			&chart,
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		tarballURL := "https://giantswarm.github.io/default-catalog/test-app-0.1.1.tgz"
		if chart.Spec.TarballURL != tarballURL {
			t.Fatalf("expected tarballURL: %#v got %#v", tarballURL, chart.Spec.TarballURL)
		}

		config.Logger.Debugf(ctx, "checked tarball URL in chart spec")
	}

	{
		config.Logger.Debugf(ctx, "checking status for app CR %#q", key.TestAppName())

		err = config.K8sClients.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: key.TestAppName(), Namespace: key.GiantSwarmNamespace()},
			&cr,
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		if cr.Status.Release.Status != helmclient.StatusDeployed {
			t.Fatalf("expected CR release status %#q got %#q", helmclient.StatusDeployed, cr.Status.Release.Status)
		}

		config.Logger.Debugf(ctx, "checked status for app CR %#q", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "deleting app CR %#q", key.TestAppName())

		err = config.K8sClients.CtrlClient().Delete(ctx, &cr)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "deleted app CR %#q", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "checking %#q release has been deleted", key.TestAppName())

		err = config.Release.WaitForReleaseStatus(ctx, key.GiantSwarmNamespace(), key.TestAppName(), helmclient.StatusUninstalled)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "checked %#q release has been deleted", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "checking chart CR %#q has been deleted", key.TestAppName())

		err = config.Release.WaitForDeletedChart(ctx, key.GiantSwarmNamespace(), key.TestAppName())
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "chart CR %#q has been deleted", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "checking app CR %#q has been deleted", key.TestAppName())

		err = config.Release.WaitForDeletedApp(ctx, key.GiantSwarmNamespace(), key.TestAppName())
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "chart CR %#q has been deleted", key.TestAppName())
	}
}
