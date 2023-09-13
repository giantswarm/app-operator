//go:build k8srequired
// +build k8srequired

package basic

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/apptest"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v6/integration/key"
)

// TestAppLifecycle tests a chart CR can be created, updated and deleted
// using a app, appCatalog CRs processed by app-operator.
//
// - Install Flux.
// - Create test App CR.
// - Ensure HelmRelease CR is deployed.
//
// - Update app version in App CR.
// - Ensure HelmRelease CR is redeployed using updated App CR information.
//
// - Delete App CR
// - Ensure HelmRelease CR is deleted.
func TestAppLifecycle(t *testing.T) {
	ctx := context.Background()

	var helmRelease helmv2.HelmRelease
	var cr v1alpha1.App
	var err error
	{
		apps := []apptest.App{
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
		config.Logger.Debugf(ctx, "checking Chart configuration in HelmRelease CR spec")

		chart := helmv2.HelmChartTemplateSpec{
			Chart:             "test-app",
			ReconcileStrategy: "ChartVersion",
			SourceRef: helmv2.CrossNamespaceObjectReference{
				Kind:      "HelmRepository",
				Name:      "default-helm-giantswarm.github.io-default-catalog",
				Namespace: "default",
			},
			Version: "0.1.0",
		}

		err = config.K8sClients.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: key.TestAppName(), Namespace: key.GiantSwarmNamespace()},
			&helmRelease,
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		if !reflect.DeepEqual(helmRelease.Spec.Chart.Spec, chart) {
			t.Fatalf("want matching specs \n %s", cmp.Diff(helmRelease.Spec.Chart.Spec, chart))
		}

		config.Logger.Debugf(ctx, "checked Chart configuration in HelmRelease CR spec")
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
		config.Logger.Debugf(ctx, "checking Chart configuration in HelmRelease CR spec")

		err = config.Release.WaitForReleaseVersion(ctx, key.GiantSwarmNamespace(), key.TestAppName(), "0.1.1")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.K8sClients.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: key.TestAppName(), Namespace: key.GiantSwarmNamespace()},
			&helmRelease,
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		chart := helmv2.HelmChartTemplateSpec{
			Chart:             "test-app",
			ReconcileStrategy: "ChartVersion",
			SourceRef: helmv2.CrossNamespaceObjectReference{
				Kind:      "HelmRepository",
				Name:      "default-helm-giantswarm.github.io-default-catalog",
				Namespace: "default",
			},
			Version: "0.1.1",
		}
		if !reflect.DeepEqual(helmRelease.Spec.Chart.Spec, chart) {
			t.Fatalf("want matching specs \n %s", cmp.Diff(helmRelease.Spec.Chart.Spec, chart))
		}

		config.Logger.Debugf(ctx, "checked Chart configuration in HelmRelease CR spec")
	}

	{
		config.Logger.Debugf(ctx, "checking status for App CR %#q", key.TestAppName())

		o := func() error {
			err = config.K8sClients.CtrlClient().Get(
				ctx,
				types.NamespacedName{Name: key.TestAppName(), Namespace: key.GiantSwarmNamespace()},
				&cr,
			)
			if err != nil {
				return microerror.Mask(err)
			}
			if cr.Status.Release.Status != helmclient.StatusDeployed {
				return fmt.Errorf("expected CR release status %#q, got %#q", helmclient.StatusDeployed, cr.Status.Release.Status)
			}
			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.Debugf(ctx, "failed to get release status '%s': retrying in %s", helmclient.StatusDeployed, t)
		}

		b := backoff.NewExponential(10*time.Minute, 60*time.Second)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "checked status for App CR %#q", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "deleting App CR %#q", key.TestAppName())

		err = config.K8sClients.CtrlClient().Delete(ctx, &cr)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "deleted App CR %#q", key.TestAppName())
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
		config.Logger.Debugf(ctx, "checking HelmRelease CR %#q has been deleted", key.TestAppName())

		err = config.Release.WaitForDeletedHelmRelease(ctx, key.GiantSwarmNamespace(), key.TestAppName())
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "HelmRelease CR %#q has been deleted", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "checking App CR %#q has been deleted", key.TestAppName())

		err = config.Release.WaitForDeletedApp(ctx, key.GiantSwarmNamespace(), key.TestAppName())
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "App CR %#q has been deleted", key.TestAppName())
	}
}
