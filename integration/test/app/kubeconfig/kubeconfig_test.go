// +build k8srequired

package kubeconfig

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2esetup/chart/env"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/app-operator/integration/key"
	"github.com/giantswarm/app-operator/integration/templates"
	"github.com/giantswarm/app-operator/pkg/label"
)

const (
	clusterName = "kind-kind"
	namespace   = "giantswarm"
)

// TestAppWithKubeconfig checks app-operator can bootstrap chart-operator
// when a kubeconfig is provided.
func TestAppWithKubeconfig(t *testing.T) {
	ctx := context.Background()
	var chartValues string
	var err error

	sampleChart := chartvalues.APIExtensionsAppE2EConfig{
		App: chartvalues.APIExtensionsAppE2EConfigApp{
			KubeConfig: chartvalues.APIExtensionsAppE2EConfigAppKubeConfig{
				InCluster: false,
				Secret: chartvalues.APIExtensionsAppE2EConfigAppConfigKubeConfigSecret{
					Name:      "kube-config",
					Namespace: namespace,
				},
			},
			Name:      key.TestAppReleaseName(),
			Namespace: namespace,
			Catalog:   key.DefaultCatalogName(),
			Version:   "0.1.0",
			Config: chartvalues.APIExtensionsAppE2EConfigAppConfig{
				ConfigMap: chartvalues.APIExtensionsAppE2EConfigAppConfigConfigMap{
					Name:      "test-app-values",
					Namespace: "default",
				},
				Secret: chartvalues.APIExtensionsAppE2EConfigAppConfigSecret{
					Name:      "test-app-secrets",
					Namespace: "default",
				},
			},
		},
		AppCatalog: chartvalues.APIExtensionsAppE2EConfigAppCatalog{
			Description: key.DefaultCatalogName(),
			Name:        key.DefaultCatalogName(),
			Title:       key.DefaultCatalogName(),
			Storage: chartvalues.APIExtensionsAppE2EConfigAppCatalogStorage{
				Type: "helm",
				URL:  key.DefaultCatalogStorageURL(),
			},
		},
		AppOperator: chartvalues.APIExtensionsAppE2EConfigAppOperator{
			Version: "1.0.0",
		},
		Namespace: namespace,
		ConfigMap: chartvalues.APIExtensionsAppE2EConfigConfigMap{
			ValuesYAML: `test:
      image:
        registry: quay.io
        repository: giantswarm/alpine-testing
        tag: 0.1.1`,
		},
		Secret: chartvalues.APIExtensionsAppE2EConfigSecret{
			ValuesYAML: `secret: "test"`,
		},
	}

	// Transform kubeconfig file to REST config and flatten.
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

		// Normally KIND assigns 127.0.0.1 as the server address. For this test
		// that should change to the Kubernetes service.
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
		config.Logger.LogCtx(ctx, "level", "debug", "message", "creating chart-operator configmap")

		_, err = config.K8sClients.K8sClient().CoreV1().ConfigMaps(namespace).Create(&corev1.ConfigMap{
			Data: map[string]string{
				"values": templates.ChartOperatorValues,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "chart-operator-config",
				Namespace: "giantswarm",
			},
		})
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "created chart-operator configmap")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "creating chart-operator app CR")

		tag, err := appcatalog.GetLatestVersion(ctx, key.DefaultCatalogStorageURL(), "chart-operator")
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		_, err = config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(namespace).Create(&v1alpha1.App{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "chart-operator",
				Namespace: "giantswarm",
				Labels: map[string]string{
					label.AppOperatorVersion: "1.0.0",
				},
			},
			Spec: v1alpha1.AppSpec{
				Catalog: "default",
				Config: v1alpha1.AppSpecConfig{
					ConfigMap: v1alpha1.AppSpecConfigConfigMap{
						Name:      "chart-operator-config",
						Namespace: "giantswarm",
					},
				},
				KubeConfig: v1alpha1.AppSpecKubeConfig{
					Secret: v1alpha1.AppSpecKubeConfigSecret{
						Name:      "kube-config",
						Namespace: "giantswarm",
					},
				},
				Name:      "chart-operator",
				Namespace: "giantswarm",
				Version:   tag,
			},
		})
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "created chart-operator app CR")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating chart value for release %#q", key.CustomResourceReleaseName()))

		chartValues, err = chartvalues.NewAPIExtensionsAppE2E(sampleChart)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created chart value for release %#q", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing release %#q", key.CustomResourceReleaseName()))

		chartInfo := release.NewStableChartInfo(key.CustomResourceReleaseName())
		err = config.Release.Install(ctx, key.CustomResourceReleaseName(), chartInfo, chartValues)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed release %#q", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for release %#q deployed", key.CustomResourceReleaseName()))

		err = config.Release.WaitForStatus(ctx, fmt.Sprintf("%s-%s", namespace, key.CustomResourceReleaseName()), "DEPLOYED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for release %#q deployed", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "waiting for chart CR created")

		err = config.Release.WaitForStatus(ctx, key.TestAppReleaseName(), "DEPLOYED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "waited for chart CR created")
	}
}
