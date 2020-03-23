// +build k8srequired

package basic

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/spf13/afero"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/helm"

	"github.com/giantswarm/app-operator/integration/key"
	"github.com/giantswarm/app-operator/integration/templates"
)

const (
	chartOperatorVersion = "chart-operator.giantswarm.io/version"
	chartOperatorRelease = "chart-operator"
	namespace            = "giantswarm"
)

// TestAppLifecycle tests a chart CR can be created, updated and deleted
// uaing a app, appCatalog CRs processed by app-operator.
//
// - Create app, appCatalog CRs using apiextensions-app-e2e-chart.
// - Ensure chart CR specified in the app CR is deployed.
//
// - Update chart CR using apiextensions-app-e2e-chart.
// - Ensure chart CR is redeployed using updated app CR information.
//
// - Delete app CR
// - Ensure chart CR is deleted.
//
func TestAppLifecycle(t *testing.T) {
	ctx := context.Background()
	var chartValues string
	var err error

	sampleChart := chartvalues.APIExtensionsAppE2EConfig{
		App: chartvalues.APIExtensionsAppE2EConfigApp{
			KubeConfig: chartvalues.APIExtensionsAppE2EConfigAppKubeConfig{
				InCluster: true,
			},
			Name:      key.TestAppReleaseName(),
			Namespace: namespace,
			Catalog:   key.DefaultCatalogName(),
			Version:   "0.1.0",
			Config: chartvalues.APIExtensionsAppE2EConfigAppConfig{
				ConfigMap: chartvalues.APIExtensionsAppE2EConfigAppConfigConfigMap{
					Name:      "test-app-values",
					Namespace: "default",
				},
				Secret: chartvalues.APIExtensionsAppE2EConfigAppConfigSecret{
					Name:      "test-app-secrets",
					Namespace: "default",
				},
			},
		},
		AppCatalog: chartvalues.APIExtensionsAppE2EConfigAppCatalog{
			Description: key.DefaultCatalogName(),
			Name:        key.DefaultCatalogName(),
			Title:       key.DefaultCatalogName(),
			Storage: chartvalues.APIExtensionsAppE2EConfigAppCatalogStorage{
				Type: "helm",
				URL:  key.DefaultCatalogStorageURL(),
			},
		},
		AppOperator: chartvalues.APIExtensionsAppE2EConfigAppOperator{
			Version: "1.0.0",
		},
		Namespace: namespace,
		ConfigMap: chartvalues.APIExtensionsAppE2EConfigConfigMap{
			ValuesYAML: `test:
      image:
        registry: quay.io
        repository: giantswarm/alpine-testing
        tag: 0.1.1`,
		},
		Secret: chartvalues.APIExtensionsAppE2EConfigSecret{
			ValuesYAML: `secret: "test"`,
		},
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing chart operator"))

		var tarballPath string
		{
			tarballURL, err := appcatalog.GetLatestChart(ctx, key.DefaultCatalogStorageURL(), "chart-operator")
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
		err = config.HelmClient.InstallReleaseFromTarball(ctx, tarballPath, namespace, helm.ReleaseName(chartOperatorRelease), helm.ValueOverrides([]byte(templates.ChartOperatorValues)))
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed chart operator"))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating chart value for release %#q", key.CustomResourceReleaseName()))

		chartValues, err = chartvalues.NewAPIExtensionsAppE2E(sampleChart)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created chart value for release %#q", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing release %#q", key.CustomResourceReleaseName()))

		chartInfo := release.NewStableChartInfo(key.CustomResourceReleaseName())
		err = config.Release.Install(ctx, key.CustomResourceReleaseName(), chartInfo, chartValues)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed release %#q", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for release %#q deployed", key.CustomResourceReleaseName()))

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", namespace, key.CustomResourceReleaseName()), "DEPLOYED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for release %#q deployed", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "waiting for chart CR created")

		err = config.Release.WaitForStatus(ctx, key.TestAppReleaseName(), "DEPLOYED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "waited for chart CR created")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "checking tarball URL in chart spec")

		tarballURL := "https://giantswarm.github.com/default-catalog/test-app-0.1.0.tgz"
		chart, err := config.K8sClients.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(key.TestAppReleaseName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		if chart.Spec.TarballURL != tarballURL {
			t.Fatalf("expected tarballURL: %#q got %#q", tarballURL, chart.Spec.TarballURL)
		}
		if chart.Labels[chartOperatorVersion] != "1.0.0" {
			t.Fatalf("expected version label: %#q got %#q", "1.0.0", chart.Labels[chartOperatorVersion])
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "checked tarball URL in chart spec")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating chart value for release %#q", key.CustomResourceReleaseName()))

		sampleChart.App.Version = "0.1.1"
		chartValues, err = chartvalues.NewAPIExtensionsAppE2E(sampleChart)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated chart value for release %#q", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating release %#q", key.CustomResourceReleaseName()))

		chartInfo := release.NewStableChartInfo(key.CustomResourceReleaseName())
		err = config.Release.Update(ctx, key.CustomResourceReleaseName(), chartInfo, chartValues)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated release %#q", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for release %#q deployed", key.CustomResourceReleaseName()))

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", namespace, key.CustomResourceReleaseName()), "DEPLOYED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for release %#q deployed", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "checking tarball URL in chart spec")

		err = config.Release.WaitForChartInfo(ctx, key.TestAppReleaseName(), "0.1.1")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		chart, err := config.K8sClients.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(key.TestAppReleaseName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		tarballURL := "https://giantswarm.github.com/default-catalog/test-app-0.1.1.tgz"
		if chart.Spec.TarballURL != tarballURL {
			t.Fatalf("expected tarballURL: %#v got %#v", tarballURL, chart.Spec.TarballURL)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "checked tarball URL in chart spec")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking status for app CR %#q", key.TestAppReleaseName()))

		cr, err := config.K8sClients.G8sClient().ApplicationV1alpha1().Apps("giantswarm").Get(key.TestAppReleaseName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		if cr.Status.Release.Status != "DEPLOYED" {
			t.Fatalf("expected CR release status %#q got %#q", "DEPLOYED", cr.Status.Release.Status)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checked status for app CR %#q", key.TestAppReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting release %#q", key.CustomResourceReleaseName()))

		err := config.Release.Delete(ctx, key.CustomResourceReleaseName())
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted release %#q", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking release %#q has been deleted", key.CustomResourceReleaseName()))

		err = config.Release.WaitForStatus(ctx, key.TestAppReleaseName(), "DELETED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("release %#q has been deleted", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking chart CR %#q has been deleted", key.ChartConfigMapName()))

		_, err = config.K8sClients.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(key.TestAppReleaseName(), metav1.GetOptions{})
		if !apierrors.IsNotFound(err) {
			t.Fatalf("expected is not found error got %#v", err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("chart CR %#q has been deleted", key.ChartConfigMapName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking chart configmap %#q has been deleted", key.ChartConfigMapName()))

		_, err = config.K8sClients.K8sClient().CoreV1().ConfigMaps(namespace).Get(key.ChartConfigMapName(), metav1.GetOptions{})
		if !apierrors.IsNotFound(err) {
			t.Fatalf("expected is not found error got %#v", err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("chart configmap %#q has been deleted", key.ChartConfigMapName()))
	}
}
