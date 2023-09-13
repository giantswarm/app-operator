//go:build k8srequired
// +build k8srequired

package workload

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/apptest"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/app-operator/v6/integration/env"
	"github.com/giantswarm/app-operator/v6/integration/key"
	"github.com/giantswarm/app-operator/v6/integration/release"
	"github.com/giantswarm/app-operator/v6/integration/templates"
	"github.com/giantswarm/app-operator/v6/pkg/project"
)

const (
	catalogConfigMapName = "default-catalog-configmap"
	clusterName          = "kind-kind"
	kubeConfigName       = "kube-config"
)

// TestWorkloadClusterMigration checks App Operator switches from Chart Operator
// to Helm Controller without any issues.
func TestWorkloadClusterMigration(t *testing.T) {
	ctx := context.Background()

	chartOperator := release.AppConfiguration{
		AppName:      key.ChartOperatorName(),
		AppNamespace: key.GiantSwarmNamespace(),
		AppValues:    templates.ChartOperatorValues,
		AppVersion:   key.ChartOperatorVersion(),
		CatalogURL:   key.DefaultCatalogStorageURL(),
	}

	err := config.Release.InstallFromTarball(ctx, chartOperator)
	if err != nil {
		t.Fatalf("expected %#v got %#v", nil, err)
	}

	{
		err = config.K8s.EnsureNamespaceCreated(ctx, key.OrganizationNamespace())
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}
	}

	// Transform kubeconfig file to restconfig and flatten.
	var kubeConfig string
	{
		c := clientcmd.GetConfigFromFileOrDie(env.KubeConfigPath())

		// Extract KIND kubeconfig settings. This is for local testing as
		// api.FlattenConfig does not work with file paths in kubeconfigs.
		clusterKubeConfig := &api.Config{
			AuthInfos: map[string]*api.AuthInfo{
				clusterName: c.AuthInfos[clusterName],
			},
			Clusters: map[string]*api.Cluster{
				clusterName: c.Clusters[clusterName],
			},
			Contexts: map[string]*api.Context{
				clusterName: c.Contexts[clusterName],
			},
		}

		err = api.FlattenConfig(clusterKubeConfig)
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		// Normally KIND assigns 127.0.0.1 as the server address. For this test
		// that should change to the Kubernetes service.
		clusterKubeConfig.Clusters[clusterName].Server = "https://kubernetes.default.svc.cluster.local"

		bytes, err := clientcmd.Write(*c)
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		// Create kubeconfig secret for the chart CR watcher in app-operator.
		secret := &corev1.Secret{
			Data: map[string][]byte{
				"value": bytes,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-kubeconfig", key.ClusterID()),
				Namespace: key.OrganizationNamespace(),
			},
		}
		_, err = config.K8sClients.K8sClient().CoreV1().Secrets(key.OrganizationNamespace()).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		kubeConfig = string(bytes)
	}

	initialAppOperator := apptest.App{
		AppCRName:      fmt.Sprintf("%s-%s", key.ClusterID(), project.Name()),
		AppCRNamespace: key.OrganizationNamespace(),
		CatalogName:    key.ControlPlaneTestCatalogName(),
		Name:           project.Name(),
		Namespace:      key.OrganizationNamespace(),
		ValuesYAML:     templates.AppOperatorCAPIWCOldValues,
		SHA:            key.AppOperatorInTestVersion(),
		WaitForDeploy:  true,
	}

	{
		apps := []apptest.App{
			initialAppOperator,
			{
				// Install test app using the workload cluster instance of
				// app-operator.
				AppCRNamespace: key.OrganizationNamespace(),
				ClusterID:      key.ClusterID(),
				CatalogName:    key.DefaultCatalogName(),
				KubeConfig:     kubeConfig,
				Name:           key.TestAppName(),
				Namespace:      metav1.NamespaceDefault,
				Version:        "0.1.0",
				WaitForDeploy:  true,
			},
		}
		err = config.AppTest.InstallApps(ctx, apps)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	// At this point the cluster runs:
	// - unique App Operator in the `giantswarm` namesapce
	// - Chart Operator in the `giantswarm` namespace
	// - App Operator in the `org-test` namespace
	// - Test App installed for the cluster from the `org-test` namespace
	//
	// What we want to do is to:
	// - install Flux App
	// - remove Chart Operator
	// - reconfigure unique App Operator by switching it to Helm Controller
	// - reconfigure `org-test` App Operator by switching it to Helm Controller
	// - check releases have been imported by Helm Controller

	// Install Flux
	{
		flux := release.AppConfiguration{
			AppName:      key.FluxAppName(),
			AppNamespace: key.FluxSystemNamespace(),
			AppValues:    "",
			AppVersion:   key.FluxAppVersion(),
			CatalogURL:   key.StableCatalogStorageHelmURL(),
		}

		err = config.Release.InstallFromTarball(ctx, flux)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	// Remove Chart Operator
	{
		err = config.HelmClient.DeleteRelease(ctx, key.GiantSwarmNamespace(), key.ChartOperatorName(), helmclient.DeleteOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForReleaseStatus(ctx, key.GiantSwarmNamespace(), key.ChartOperatorName(), helmclient.StatusUninstalled)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	// Reconfigure unique App Operator
	{
		appOperator := release.AppConfiguration{
			AppName:      project.Name(),
			AppNamespace: key.GiantSwarmNamespace(),
			AppValues:    templates.AppOperatorCAPIValues,
			AppVersion:   key.AppOperatorInTestVersion(),
			CatalogURL:   key.ControlPlaneTestCatalogStorageURL(),
		}

		err = config.Release.UpdateFromTarball(ctx, appOperator)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	// At this point the unique App Operator should be using the Helm Controller, hence the `org-test` App Operator
	// App CR should be installed with it instead of Chart Operator. We verify that by making sure the Chart CR is
	// gone in result of running `migration` resource of unique App Operator, and we also make sure the HelmRelease CR
	// is present in the `org-test` namespace.
	{
		err = config.Release.WaitForDeletedChart(
			ctx,
			key.GiantSwarmNamespace(),
			fmt.Sprintf("%s-%s", key.ClusterID(), project.Name()),
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForHelmRelease(
			ctx,
			key.OrganizationNamespace(),
			fmt.Sprintf("%s-%s", key.ClusterID(), project.Name()),
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForReleaseRevision(
			ctx, key.OrganizationNamespace(),
			fmt.Sprintf("%s-%s", key.ClusterID(), project.Name()),
			2,
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForAppStatusRelease(
			ctx, key.OrganizationNamespace(),
			fmt.Sprintf("%s-%s", key.ClusterID(),
				project.Name()),
			v1alpha1.AppStatusRelease{
				Reason: "Helm upgrade succeeded",
				Status: helmclient.StatusDeployed,
			},
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	// Update `org-test` App Operator App CR by enabling Helm Controller backend
	{
		cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.OrganizationNamespace()).Get(
			ctx,
			fmt.Sprintf("%s-user-values", project.Name()),
			metav1.GetOptions{},
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		cm.Data = map[string]string{
			"values": templates.AppOperatorCAPIWCNewValues,
		}

		_, err = config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.OrganizationNamespace()).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForReleaseRevision(
			ctx, key.OrganizationNamespace(),
			fmt.Sprintf("%s-%s", key.ClusterID(), project.Name()),
			3,
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForAppStatusRelease(
			ctx, key.OrganizationNamespace(),
			fmt.Sprintf("%s-%s", key.ClusterID(),
				project.Name()),
			v1alpha1.AppStatusRelease{
				Reason: "Helm upgrade succeeded",
				Status: helmclient.StatusDeployed,
			},
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	// Make sure the Test app has been migrated
	{
		err = config.Release.WaitForHelmRelease(
			ctx,
			key.OrganizationNamespace(),
			key.TestAppName(),
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForReleaseRevision(
			ctx,
			key.DefaultNamespace(),
			key.TestAppName(),
			2,
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		err = config.Release.WaitForDeletedChart(
			ctx,
			key.GiantSwarmNamespace(),
			key.TestAppName(),
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}
}
