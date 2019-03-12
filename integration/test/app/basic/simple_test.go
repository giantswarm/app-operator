// +build k8srequired

package basic

import (
	"fmt"
	"golang.org/x/net/context"
	"testing"

	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/integration/ensure"
)

const (
	namespace                 = "giantswarm"
	customResourceReleaseName = "apiextensions-app-e2e-chart"
	chartOperatorVersion      = "chart-operator.giantswarm.io/version"
	testAppReleaseName        = "test-app"
	testAppCatalogReleaseName = "test-app-catalog"
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
	var originalResourceVersion string

	sampleChart := chartvalues.APIExtensionsAppE2EConfig{
		App: chartvalues.APIExtensionsAppE2EConfigApp{
			Name:      testAppReleaseName,
			Namespace: namespace,
			Catalog:   testAppCatalogReleaseName,
			Version:   "1.0.0",
		},
		AppCatalog: chartvalues.APIExtensionsAppE2EConfigAppCatalog{
			Name:  testAppCatalogReleaseName,
			Title: testAppCatalogReleaseName,
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

	// Test creation.
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating chart %#q", customResourceReleaseName))

		chartValues, err := chartvalues.NewAPIExtensionsAppE2E(sampleChart)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		chartInfo := release.NewStableChartInfo(customResourceReleaseName)
		err = config.Release.Install(ctx, customResourceReleaseName, chartInfo, chartValues)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", namespace, customResourceReleaseName), "DEPLOYED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created chart %#q", customResourceReleaseName))

		tarballURL := "https://giantswarm.github.com/sample-catalog/test-app-1.0.0.tgz"
		err = ensure.WaitForUpdatedChartCR(ctx, ensure.Create, &config, namespace, testAppReleaseName, "")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		chart, err := config.Host.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(testAppReleaseName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		if chart.Spec.TarballURL != tarballURL {
			t.Fatalf("expected tarballURL: %#q got %#q", tarballURL, chart.Spec.TarballURL)
		}
		if chart.Labels[chartOperatorVersion] != "1.0.0" {
			t.Fatalf("expected version label: %#q got %#q", "1.0.0", chart.Labels[chartOperatorVersion])
		}
		originalResourceVersion = chart.ObjectMeta.ResourceVersion
	}

	// Test update
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating chart %#q", customResourceReleaseName))

		sampleChart.App.Version = "1.0.1"

		chartValues, err := chartvalues.NewAPIExtensionsAppE2E(sampleChart)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		chartInfo := release.NewStableChartInfo(customResourceReleaseName)
		err = config.Release.Update(ctx, customResourceReleaseName, chartInfo, chartValues)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", namespace, customResourceReleaseName), "DEPLOYED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated chart %#q", customResourceReleaseName))

		tarballURL := "https://giantswarm.github.com/sample-catalog/test-app-1.0.1.tgz"
		err = ensure.WaitForUpdatedChartCR(ctx, ensure.Update, &config, namespace, testAppReleaseName, originalResourceVersion)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		chart, err := config.Host.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(testAppReleaseName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		if chart.Spec.TarballURL != tarballURL {
			t.Fatalf("expected tarballURL: %#v got %#v", tarballURL, chart.Spec.TarballURL)
		}
	}

	// Test deletion
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting chart %#q", customResourceReleaseName))

		err := config.Release.Delete(ctx, customResourceReleaseName)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", namespace, customResourceReleaseName), "DELETED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted chart %#q", customResourceReleaseName))

		err = ensure.WaitForUpdatedChartCR(ctx, ensure.Delete, &config, namespace, testAppReleaseName, "")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}
}
