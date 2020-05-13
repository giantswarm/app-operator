// +build k8srequired

package basic

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/helmclient"
	"github.com/spf13/afero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/integration/key"
	"github.com/giantswarm/app-operator/pkg/label"
)

const (
	chartOperatorVersion = "chart-operator.giantswarm.io/version"
	chartOperatorRelease = "chart-operator"
	namespace            = "giantswarm"
)

// TestAppLifecycle tests a chart CR can be created, updated and deleted
// uaing a app, appCatalog CRs processed by app-operator.
//
// - Create appcatalog and app CRs.
// - Install chart-operator.
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
	var err error

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing chart operator"))

		var tarballPath string
		{
			tarballURL, err := appcatalog.GetLatestChart(ctx, key.DefaultCatalogStorageURL(), chartOperatorRelease, "")
			if err != nil {
				t.Fatalf("expected %#v got %#v", nil, err)
			}

			// TODO: Removing hardcoding once there is a chart-operator release
			// with Helm 3 support in the default catalog.
			tarballURL = "https://giantswarm.github.io/default-test-catalog/chart-operator-0.12.1-13521d4e2cb5378dbff26995e094d1c23a15e121.tgz"

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

		opts := helmclient.InstallOptions{
			ReleaseName: chartOperatorRelease,
		}
		values := map[string]interface{}{
			"clusterDNSIP": "10.96.0.10",
			"e2e":          "true",
		}
		err = config.HelmClient.InstallReleaseFromTarball(ctx, tarballPath, key.Namespace(), values, opts)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed chart operator"))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for chart-operator pod"))

		err = config.Release.WaitForPod(ctx, namespace, "app=chart-operator")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for chart-operator pod"))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating %#q appcatalog cr", key.DefaultCatalogName()))

		appCatalogCR := &v1alpha1.AppCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: key.DefaultCatalogName(),
				Labels: map[string]string{
					label.AppOperatorVersion: key.AppOperatorVersion(),
					// Helm major version is 3 for the master branch.
					label.HelmMajorVersion: "3",
				},
			},
			Spec: v1alpha1.AppCatalogSpec{
				Description: key.DefaultCatalogName(),
				Title:       key.DefaultCatalogName(),
				Storage: v1alpha1.AppCatalogSpecStorage{
					Type: "helm",
					URL:  key.DefaultCatalogStorageURL(),
				},
			},
		}
		_, err = config.K8sClients.G8sClient().ApplicationV1alpha1().AppCatalogs().Create(appCatalogCR)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created %#q appcatalog cr", key.DefaultCatalogName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating %#q app cr", key.TestAppReleaseName()))

		appCR := &v1alpha1.App{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.TestAppReleaseName(),
				Namespace: namespace,
				Labels: map[string]string{
					label.AppOperatorVersion: key.AppOperatorVersion(),
				},
			},
			Spec: v1alpha1.AppSpec{
				Catalog: key.DefaultCatalogName(),
				KubeConfig: v1alpha1.AppSpecKubeConfig{
					InCluster: true,
				},
				Name:      key.TestAppReleaseName(),
				Namespace: namespace,
				Version:   "0.1.0",
			},
		}
		_, err = config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(key.Namespace()).Create(appCR)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating %#q app cr", key.TestAppReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "waiting for chart CR created")

		err = config.Release.WaitForReleaseStatus(ctx, namespace, key.TestAppReleaseName(), helmclient.StatusDeployed)
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
		if chart.Labels[label.ChartOperatorVersion] != "1.0.0" {
			t.Fatalf("expected version label: %#q got %#q", "1.0.0", chart.Labels[chartOperatorVersion])
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "checked tarball URL in chart spec")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating app %#q", key.TestAppReleaseName()))

		cr, err := config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(key.Namespace()).Get(key.TestAppReleaseName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		cr.Spec.Version = "0.1.1"
		_, err = config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(key.Namespace()).Update(cr)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated app %#q", key.TestAppReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "checking tarball URL in chart spec")

		err = config.Release.WaitForReleaseVersion(ctx, namespace, key.TestAppReleaseName(), "0.1.1")
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
		if cr.Status.Release.Status != helmclient.StatusDeployed {
			t.Fatalf("expected CR release status %#q got %#q", helmclient.StatusDeployed, cr.Status.Release.Status)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checked status for app CR %#q", key.TestAppReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting app CR %#q", key.TestAppReleaseName()))

		err = config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(namespace).Delete(key.TestAppReleaseName(), &metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted app CR %#q", key.TestAppReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking %#q release has been deleted", key.TestAppReleaseName()))

		err = config.Release.WaitForReleaseStatus(ctx, namespace, key.TestAppReleaseName(), helmclient.StatusUninstalled)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checked %#q release has been deleted", key.TestAppReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking chart CR %#q has been deleted", key.TestAppReleaseName()))

		err = config.Release.WaitForDeletedChart(ctx, namespace, key.TestAppReleaseName())
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("chart CR %#q has been deleted", key.TestAppReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking app CR %#q has been deleted", key.TestAppReleaseName()))

		err = config.Release.WaitForDeletedApp(ctx, namespace, key.TestAppReleaseName())
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("chart CR %#q has been deleted", key.TestAppReleaseName()))
	}
}
