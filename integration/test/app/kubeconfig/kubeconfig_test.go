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
)

const (
	namespace            = "giantswarm"
	chartOperatorVersion = "chart-operator.giantswarm.io/version"
	targetNamespace      = "test"
	testAppCatalogName   = "test-app-catalog"
)

// TestAppLifecycleUsingKubeconfig perform same tests as TestAppLifeCycle except it using kubeConfig spec
func TestAppLifecycleUsingKubeconfig(t *testing.T) {
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
			Catalog:   testAppCatalogName,
			Version:   "0.7.1",
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
			Name:  testAppCatalogName,
			Title: testAppCatalogName,
			Storage: chartvalues.APIExtensionsAppE2EConfigAppCatalogStorage{
				Type: "helm",
				URL:  "https://giantswarm.github.com/sample-catalog",
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

	// Transform kubeconfig file to restconfig and flatten
	var bytes []byte
	{
		c := clientcmd.GetConfigFromFileOrDie(env.KubeConfigPath())

		err = api.FlattenConfig(c)
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		// Normally KIND assign 127.0.0.1 as server address, that should change into kubernetes
		c.Clusters["kind-kind"].Server = "https://kubernetes.default.svc.cluster.local"

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
		config.Logger.LogCtx(ctx, "level", "debug", "message", "creating chart-operator app CR")

		_, err = config.K8sClients.K8sClient().CoreV1().ConfigMaps(namespace).Create(&corev1.ConfigMap{
			Data: map[string]string{
				"values": templates.ChartOperatorValues,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default-catalog-config",
				Namespace: "giantswarm",
			},
		})
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		c, err := config.K8sClients.G8sClient().ApplicationV1alpha1().AppCatalogs().Create(&v1alpha1.AppCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
			Spec: v1alpha1.AppCatalogSpec{
				Config: v1alpha1.AppCatalogSpecConfig{
					ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
						Name:      "default-catalog-config",
						Namespace: "giantswarm",
					},
				},
				Storage: v1alpha1.AppCatalogSpecStorage{
					Type: "helm",
					URL:  key.DefaultCatalogStorageURL(),
				},
				Title: "Giant Swarm Default Catalog",
			},
		})
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		tag, err := appcatalog.GetLatestVersion(ctx, c.Spec.Storage.URL, "chart-operator")
		if err != nil {
			t.Fatalf("expected nil got %#v", err)
		}

		_, err = config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(namespace).Create(&v1alpha1.App{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "chart-operator",
				Namespace: "giantswarm",
				Labels: map[string]string{
					"app-operator.giantswarm.io/version": "1.0.0",
				},
			},
			Spec: v1alpha1.AppSpec{
				Catalog: "default",
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

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "checking tarball URL in chart spec")

		tarballURL := "https://giantswarm.github.com/sample-catalog/kubernetes-test-app-chart-0.6.7.tgz"
		chart, err := config.K8sClients.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(key.TestAppReleaseName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
		if chart.Spec.TarballURL != tarballURL {
			t.Fatalf("expected tarballURL: %#q got %#q", tarballURL, chart.Spec.TarballURL)
		}
		if chart.Labels[chartOperatorVersion] != "1.0.0" {
			t.Fatalf("expected version label: %#q got %#q", "1.0.0", chart.Labels[chartOperatorVersion])
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "checked tarball URL in chart spec")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating chart value for release %#q", key.CustomResourceReleaseName()))

		sampleChart.App.Version = "0.6.8"
		chartValues, err = chartvalues.NewAPIExtensionsAppE2E(sampleChart)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated chart value for release %#q", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating release %#q", key.CustomResourceReleaseName()))

		chartInfo := release.NewStableChartInfo(key.CustomResourceReleaseName())
		err = config.Release.Update(ctx, key.CustomResourceReleaseName(), chartInfo, chartValues)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated release %#q", key.CustomResourceReleaseName()))
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
		config.Logger.LogCtx(ctx, "level", "debug", "message", "checking tarball URL in chart spec")

		err = config.Release.WaitForChartInfo(ctx, key.TestAppReleaseName(), "0.6.8")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		chart, err := config.K8sClients.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(key.TestAppReleaseName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		tarballURL := "https://giantswarm.github.com/sample-catalog/kubernetes-test-app-chart-0.6.8.tgz"
		if chart.Spec.TarballURL != tarballURL {
			t.Fatalf("expected tarballURL: %#v got %#v", tarballURL, chart.Spec.TarballURL)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "checked tarball URL in chart spec")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting release %#q", key.CustomResourceReleaseName()))

		err := config.Release.Delete(ctx, key.CustomResourceReleaseName())
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted release %#q", key.CustomResourceReleaseName()))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "checking chart CR had been deleted")

		err = config.Release.WaitForStatus(ctx, key.TestAppReleaseName(), "DELETED")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "checked chart CR had been deleted")
	}
}
