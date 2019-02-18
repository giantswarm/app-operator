// +build k8srequired

package basic

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/microerror"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	namespace                 = "giantswarm"
	customResourceReleaseName = "apiextensions-app-e2e-chart"
	testAppReleaseName        = "test-app"
	testAppCatalogReleaseName = "test-app-catalog"
)

type CRTestCase int

const (
	create CRTestCase = 0
	update CRTestCase = 1
	delete CRTestCase = 2
)

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
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating app %#q", customResourceReleaseName))

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

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created app %#q", customResourceReleaseName))

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking chart CR %#q is deployed", testAppReleaseName))

		tarballURL := "https://giantswarm.github.com/sample-catalog/test-app-1.0.0.tgz"
		err = waitForChartUpdated(ctx, create, "")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		chart, err := config.Host.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(testAppReleaseName, v1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		if !reflect.DeepEqual(chart.Spec.TarballURL, tarballURL) {
			t.Fatalf("expected tarballURL: %#v got %#v", tarballURL, chart.Spec.TarballURL)
		}
		originalResourceVersion = chart.ObjectMeta.ResourceVersion
	}

	// Test update
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating app %#q", customResourceReleaseName))

		sampleChart.App.Version = "1.0.1"
		sampleChart.AppCatalog.Storage.URL = "https://giantswarm.github.com/sample-catalog_1/"

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

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated app %#q", customResourceReleaseName))

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking chart CR %#q is updated", testAppReleaseName))

		tarballURL := "https://giantswarm.github.com/sample-catalog_1/test-app-1.0.1.tgz"
		err = waitForChartUpdated(ctx, update, originalResourceVersion)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		chart, err := config.Host.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(testAppReleaseName, v1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		if !reflect.DeepEqual(chart.Spec.TarballURL, tarballURL) {
			t.Fatalf("expected tarballURL: %#v got %#v", tarballURL, chart.Spec.TarballURL)
		}
	}

	// Test deletion
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting app %#q", customResourceReleaseName))

		err := config.Release.Delete(ctx, customResourceReleaseName)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", namespace, customResourceReleaseName), "DELETED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted app %#q", customResourceReleaseName))

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking chart CR %#q is deleted", testAppReleaseName))

		err = waitForChartUpdated(ctx, delete, "")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}
}

// searchChart will find Chart CR which have name as testAppReleaseName and resourceVersion greater than one we have.
func waitForChartUpdated(ctx context.Context, cases CRTestCase, resourceVersion string) error {
	operation := func() error {
		chart, err := config.Host.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(testAppReleaseName, v1.GetOptions{})
		switch cases {
		case create:
			if err != nil {
				return microerror.Mask(err)
			}
			return nil
		case update:
			if err != nil {
				return microerror.Mask(err)
			}
			if chart.ObjectMeta.ResourceVersion < resourceVersion {
				return microerror.Mask(testError)
			}
		case delete:
			if errors.IsNotFound(err) {
				return nil
			} else {
				return microerror.Mask(err)
			}
		}
		return nil
	}
	notify := func(err error, t time.Duration) {
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("failed to detect the changed in chart CR: retrying in %s", t))
	}
	b := backoff.NewExponential(3*time.Minute, 10*time.Second)
	err := backoff.RetryNotify(operation, b, notify)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
