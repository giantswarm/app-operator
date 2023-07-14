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

	"github.com/giantswarm/app-operator/v6/integration/env"
	"github.com/giantswarm/app-operator/v6/integration/key"
	"github.com/giantswarm/app-operator/v6/integration/templates"
	"github.com/giantswarm/app-operator/v6/pkg/project"
)

const (
	catalogConfigMapName = "default-catalog-configmap"
	clusterName          = "kind-kind"
	kubeConfigName       = "kube-config"
)

// TestWorkloadCluster checks App Operator works as expected with
// HelmController for a workload cluster.
func TestWorkloadCluster(t *testing.T) {
	ctx := context.Background()
	var err error

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

	{
		apps := []apptest.App{
			{
				// Install app-operator in the organization namespace.
				AppCRName:      fmt.Sprintf("%s-%s", key.ClusterID(), project.Name()),
				AppCRNamespace: key.OrganizationNamespace(),
				CatalogName:    key.ControlPlaneTestCatalogName(),
				Name:           project.Name(),
				Namespace:      key.OrganizationNamespace(),
				ValuesYAML:     templates.AppOperatorCAPIWCNewValues,
				SHA:            key.AppOperatorInTestVersion(),
				WaitForDeploy:  true,
			},
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
}
