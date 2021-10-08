//go:build k8srequired
// +build k8srequired

package workload

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/apptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/app-operator/v5/integration/env"
	"github.com/giantswarm/app-operator/v5/integration/key"
	"github.com/giantswarm/app-operator/v5/integration/templates"
	"github.com/giantswarm/app-operator/v5/pkg/project"
)

const (
	catalogConfigMapName = "default-catalog-configmap"
	clusterName          = "kind-kind"
	kubeConfigName       = "kube-config"
)

// TestWorkloadCluster checks app-operator can bootstrap chart-operator
// when a kubeconfig is provided.
func TestWorkloadCluster(t *testing.T) {
	ctx := context.Background()
	var err error

	{
		err = config.K8s.EnsureNamespaceCreated(ctx, key.WorkloadClusterNamespace())
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
				"kubeConfig": bytes,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-kubeconfig", key.WorkloadClusterNamespace()),
				Namespace: key.WorkloadClusterNamespace(),
			},
		}
		_, err = config.K8sClients.K8sClient().CoreV1().Secrets(key.WorkloadClusterNamespace()).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		kubeConfig = string(bytes)
	}

	{
		apps := []apptest.App{
			{
				// Bootstrap chart-operator in the giantswarm namespace.
				AppCRName:     key.ChartOperatorName(),
				CatalogName:   key.ControlPlaneCatalogName(),
				KubeConfig:    kubeConfig,
				Name:          key.ChartOperatorName(),
				Namespace:     key.GiantSwarmNamespace(),
				ValuesYAML:    templates.ChartOperatorValues,
				Version:       key.ChartOperatorVersion(),
				WaitForDeploy: true,
			},
			{
				// Install app-operator in the workload cluster namespace.
				AppCRName:      fmt.Sprintf("%s-%s", project.Name(), key.WorkloadClusterNamespace()),
				AppCRNamespace: key.WorkloadClusterNamespace(),
				CatalogName:    key.ControlPlaneTestCatalogName(),
				Name:           project.Name(),
				Namespace:      key.WorkloadClusterNamespace(),
				ValuesYAML:     templates.AppOperatorValues,
				SHA:            env.CircleSHA(),
				WaitForDeploy:  true,
			},
			{
				// Install test app using the workload cluster instance of
				// app-operator.
				AppCRNamespace:     key.WorkloadClusterNamespace(),
				AppOperatorVersion: project.Version(),
				CatalogName:        key.DefaultCatalogName(),
				KubeConfig:         kubeConfig,
				Name:               key.TestAppName(),
				Namespace:          metav1.NamespaceDefault,
				Version:            "0.1.0",
				WaitForDeploy:      true,
			},
		}
		err = config.AppTest.InstallApps(ctx, apps)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}
}
