// +build k8srequired

package app

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
	giantswarm                = "giantswarm"
	customResourceReleaseName = "apiextensions-app-e2e-chart"
	testAppReleaseName        = "test-app"
	testAppCatalogReleaseName = "test-app-catalog"
)

func TestAppLifecycle(t *testing.T) {
	ctx := context.Background()

	// Test creation.
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating app %#q", customResourceReleaseName))

		c := chartvalues.APIExtensionsAppE2EConfig{
			App: chartvalues.APIExtensionsAppE2EConfigApp{
				Name:      testAppReleaseName,
				Namespace: giantswarm,
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
			Namespace: giantswarm,
		}

		chartValues, err := chartvalues.NewAPIExtensionsAppE2E(c)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		chartInfo := release.NewStableChartInfo(customResourceReleaseName)
		err = config.Release.Install(ctx, customResourceReleaseName, chartInfo, chartValues)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", giantswarm, customResourceReleaseName), "DEPLOYED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created app %#q", customResourceReleaseName))

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking chart CR %#q is deployed", testAppReleaseName))

		tarballURL := "https://giantswarm.github.com/sample-catalog/test-app-1.0.0.tgz"
		operation := func() error {
			chart, err := config.Host.G8sClient().ApplicationV1alpha1().Charts(giantswarm).Get(testAppReleaseName, v1.GetOptions{})
			if err != nil {
				return microerror.Maskf(err, fmt.Sprintf("expected %#v got %#v", nil, err))
			}
			if !reflect.DeepEqual(chart.Spec.TarballURL, tarballURL) {
				return microerror.Maskf(notMatching, fmt.Sprintf("expected tarballURL: %#v got %#v", tarballURL, chart.Spec.TarballURL))
			}
			return nil
		}
		notify := func(err error, t time.Duration) {
			config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("failed to get created chart: retrying in %s", t))
		}
		b := backoff.NewExponential(30*time.Second, 10*time.Second)
		err = backoff.RetryNotify(operation, b, notify)
		if err != nil {
			t.Fatalf("%s", err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("chart %#q is deployed", testAppReleaseName))
	}

	// Test update
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating app %#q", customResourceReleaseName))

		c := chartvalues.APIExtensionsAppE2EConfig{
			App: chartvalues.APIExtensionsAppE2EConfigApp{
				Name:      testAppReleaseName,
				Namespace: giantswarm,
				Catalog:   testAppCatalogReleaseName,
				Version:   "1.0.1",
			},
			AppCatalog: chartvalues.APIExtensionsAppE2EConfigAppCatalog{
				Name:  testAppCatalogReleaseName,
				Title: testAppCatalogReleaseName,
				Storage: chartvalues.APIExtensionsAppE2EConfigAppCatalogStorage{
					Type: "helm",
					URL:  "https://giantswarm.github.com/sample-catalog_1/",
				},
			},
			AppOperator: chartvalues.APIExtensionsAppE2EConfigAppOperator{
				Version: "1.0.0",
			},
			Namespace: giantswarm,
		}

		chartValues, err := chartvalues.NewAPIExtensionsAppE2E(c)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		chartInfo := release.NewStableChartInfo(customResourceReleaseName)
		err = config.Release.Update(ctx, customResourceReleaseName, chartInfo, chartValues)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", giantswarm, customResourceReleaseName), "DEPLOYED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated app %#q", customResourceReleaseName))

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking chart CR %#q is updated", testAppReleaseName))

		tarballURL := "https://giantswarm.github.com/sample-catalog_1/test-app-1.0.1.tgz"
		operation := func() error {
			chart, err := config.Host.G8sClient().ApplicationV1alpha1().Charts(giantswarm).Get(testAppReleaseName, v1.GetOptions{})
			if err != nil {
				return microerror.Maskf(err, fmt.Sprintf("expected %#v got %#v", nil, err))
			}
			if !reflect.DeepEqual(chart.Spec.TarballURL, tarballURL) {
				return microerror.Maskf(notMatching, fmt.Sprintf("expected tarballURL: %#v got %#v", tarballURL, chart.Spec.TarballURL))
			}
			return nil
		}
		notify := func(err error, t time.Duration) {
			config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("failed to get updated chart: retrying in %s", t))
		}
		b := backoff.NewExponential(1*time.Minute, 10*time.Second)
		err = backoff.RetryNotify(operation, b, notify)
		if err != nil {
			t.Fatalf("%s", err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("chart CR %#q is updated", testAppReleaseName))
	}

	// Test deletion
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting app %#q", customResourceReleaseName))

		err := config.Release.Delete(ctx, customResourceReleaseName)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", giantswarm, customResourceReleaseName), "DELETED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted app %#q", customResourceReleaseName))

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking chart CR %#q is deleted", testAppReleaseName))

		operation := func() error {
			_, err = config.Host.G8sClient().ApplicationV1alpha1().Charts(giantswarm).Get(testAppReleaseName, v1.GetOptions{})
			if errors.IsNotFound(err) {
				return nil
			} else if err != nil {
				return microerror.Mask(err)
			}
			return microerror.Mask(notDeleted)
		}
		notify := func(err error, t time.Duration) {
			config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("failed to delete chart: retrying in %s", t))
		}
		b := backoff.NewExponential(1*time.Minute, 10*time.Second)
		err = backoff.RetryNotify(operation, b, notify)
		if err != nil {
			t.Fatalf("%s", err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("chart CR %#q is deleted", testAppReleaseName))
	}
}
