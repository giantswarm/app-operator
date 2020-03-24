// +build k8srequired

package kubeconfig

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/e2esetup/chart/env"
	"github.com/giantswarm/helmclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/app-operator/integration/key"
	"github.com/giantswarm/app-operator/integration/templates"
	"github.com/giantswarm/app-operator/pkg/label"
)

const (
	catalogConfigMapName = "default-catalog-configmap"
	chartOperatorName    = "chart-operator"
	clusterName          = "kind-kind"
	namespace            = "giantswarm"
)

// TestAppWithKubeconfig checks app-operator can bootstrap chart-operator
// when a kubeconfig is provided.
func TestAppWithKubeconfig(t *testing.T) {
	ctx := context.Background()
	var err error

	// Transform kubeconfig file to restconfig and flatten.
	var bytes []byte
	{
		c := clientcmd.GetConfigFromFileOrDie(env.KubeConfigPath())

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

		// Normally KIND assign 127.0.0.1 as server address, that should change into kubernetes
		clusterKubeConfig.Clusters[clusterName].Server = "https://kubernetes.default.svc.cluster.local"

		bytes, err = clientcmd.Write(*c)
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "creating kubeconfig secret")

		_, err = config.K8sClients.K8sClient().CoreV1().Secrets(namespace).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-config",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"kubeConfig": bytes,
			},
		})
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "created kubeconfig secret")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "creating catalog configmap")

		_, err = config.K8sClients.K8sClient().CoreV1().ConfigMaps(namespace).Create(&corev1.ConfigMap{
			Data: map[string]string{
				"values": templates.ChartOperatorValues,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      catalogConfigMapName,
				Namespace: namespace,
			},
		})
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "creating catalog configmap")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating %#q appcatalog cr", key.DefaultCatalogName()))

		appCatalogCR := &v1alpha1.AppCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: key.DefaultCatalogName(),
				Labels: map[string]string{
					label.AppOperatorVersion: "1.0.0",
				},
			},
			Spec: v1alpha1.AppCatalogSpec{
				Config: v1alpha1.AppCatalogSpecConfig{
					ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
						Name:      catalogConfigMapName,
						Namespace: namespace,
					},
				},
				Description: key.DefaultCatalogName(),
				Storage: v1alpha1.AppCatalogSpecStorage{
					Type: "helm",
					URL:  key.DefaultCatalogStorageURL(),
				},
				Title: key.DefaultCatalogName(),
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
					label.AppOperatorVersion: "1.0.0",
				},
			},
			Spec: v1alpha1.AppSpec{
				Catalog: key.DefaultCatalogName(),
				KubeConfig: v1alpha1.AppSpecKubeConfig{
					Context: v1alpha1.AppSpecKubeConfigContext{
						Name: clusterName,
					},
					InCluster: false,
					Secret: v1alpha1.AppSpecKubeConfigSecret{
						Name:      "kube-config",
						Namespace: namespace,
					},
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
		config.Logger.LogCtx(ctx, "level", "debug", "message", "creating chart-operator app CR")

		tag, err := appcatalog.GetLatestVersion(ctx, key.DefaultCatalogStorageURL(), "chart-operator")
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		_, err = config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(namespace).Create(&v1alpha1.App{
			ObjectMeta: metav1.ObjectMeta{
				Name:      chartOperatorName,
				Namespace: namespace,
				Labels: map[string]string{
					label.AppOperatorVersion: "1.0.0",
				},
			},
			Spec: v1alpha1.AppSpec{
				Catalog: "default",
				KubeConfig: v1alpha1.AppSpecKubeConfig{
					Context: v1alpha1.AppSpecKubeConfigContext{
						Name: clusterName,
					},
					InCluster: false,
					Secret: v1alpha1.AppSpecKubeConfigSecret{
						Name:      "kube-config",
						Namespace: namespace,
					},
				},
				Name:      chartOperatorName,
				Namespace: namespace,
				Version:   tag,
			},
		})
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "created chart-operator app CR")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for release %#q deployed", chartOperatorName))

		err = config.Release.WaitForStatus(ctx, namespace, chartOperatorName, helmclient.StatusDeployed)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for release %#q deployed", chartOperatorName))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for release %#q deployed", key.TestAppReleaseName()))

		err = config.Release.WaitForStatus(ctx, namespace, key.TestAppReleaseName(), helmclient.StatusDeployed)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for release %#q deployed", key.TestAppReleaseName()))
	}
}
