// +build k8srequired

package basic

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/integration/key"
)

const (
	namespace            = "giantswarm"
	chartOperatorVersion = "chart-operator.giantswarm.io/version"
	testAppCatalogName   = "test-app-catalog"
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
// - Delete apiextensions-app-e2e-chart.
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
			Catalog:   testAppCatalogName,
			Version:   "0.6.7",
		},
		AppCatalog: chartvalues.APIExtensionsAppE2EConfigAppCatalog{
			Name:  testAppCatalogName,
			Title: testAppCatalogName,
			Storage: chartvalues.APIExtensionsAppE2EConfigAppCatalogStorage{
				Type: "helm",
				URL:  "https://giantswarm.github.com/sample-catalog",
			},
		},
		AppOperator: chartvalues.APIExtensionsAppE2EConfigAppOperator{
			Version: "1.0.0",
		},
		Namespace: namespace,
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

		tarballURL := "https://giantswarm.github.com/sample-catalog/kubernetes-test-app-chart-0.6.7.tgz"
		chart, err := config.Host.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(key.TestAppReleaseName(), metav1.GetOptions{})
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

		sampleChart.App.Version = "0.6.8"
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

		err = config.Release.WaitForChartInfo(ctx, key.TestAppReleaseName(), "0.6.8")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		chart, err := config.Host.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(key.TestAppReleaseName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		tarballURL := "https://giantswarm.github.com/sample-catalog/kubernetes-test-app-chart-0.6.8.tgz"
		if chart.Spec.TarballURL != tarballURL {
			t.Fatalf("expected tarballURL: %#v got %#v", tarballURL, chart.Spec.TarballURL)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "checked tarball URL in chart spec")
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
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for release %#q deleted", key.CustomResourceReleaseName()))

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", namespace, key.CustomResourceReleaseName()), "DELETED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for release %#q deleted", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "checking chart CR had been deleted")

		err = config.Release.WaitForStatus(ctx, key.TestAppReleaseName(), "DELETED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "checked chart CR had been deleted")
	}
}
