// +build k8srequired

package workload

import (
	"context"
	"testing"

	"github.com/giantswarm/apptest"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/app-operator/v3/integration/env"
	"github.com/giantswarm/app-operator/v3/integration/key"
	"github.com/giantswarm/app-operator/v3/integration/templates"
	"github.com/giantswarm/app-operator/v3/pkg/project"
)

const (
	catalogConfigMapName = "default-catalog-configmap"
	clusterName          = "kind-kind"
	kubeConfigName       = "kube-config"
)

// TestWorkloadClusterBootstrap checks app-operator-unique can bootstrap an
// app-operator for the workload cluster and that is can install a test app.
func TestWorkloadCluster(t *testing.T) {
	ctx := context.Background()
	var err error

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

		kubeConfig = string(bytes)
	}

	{
		n := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: key.WorkloadClusterNamespace(),
			},
		}
		_, err := config.AppTest.K8sClient().CoreV1().Namespaces().Create(ctx, n, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			t.Logf("namespace %#q already exists", key.WorkloadClusterNamespace())
			// fall through
		} else if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}
	}

	{
		apps := []apptest.App{
			{
				// Bootstrap chart-operator in the giantswarm namespace.
				AppCRNamespace:     key.Namespace(),
				AppOperatorVersion: project.ManagementClusterAppVersion(),
				CatalogName:        key.DefaultCatalogName(),
				Name:               key.ChartOperatorName(),
				Namespace:          key.Namespace(),
				KubeConfig:         kubeConfig,
				ValuesYAML:         templates.ChartOperatorValues,
				Version:            key.ChartOperatorVersion(),
				WaitForDeploy:      true,
			},
			{
				// Install app-operator in the workload cluster namespace.
				AppCRNamespace:     key.WorkloadClusterNamespace(),
				AppOperatorVersion: project.ManagementClusterAppVersion(),
				CatalogName:        key.ControlPlaneTestCatalogName(),
				Name:               project.Name(),
				Namespace:          key.WorkloadClusterNamespace(),
				ValuesYAML:         templates.AppOperatorValues,
				SHA:                env.CircleSHA(),
				WaitForDeploy:      true,
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
