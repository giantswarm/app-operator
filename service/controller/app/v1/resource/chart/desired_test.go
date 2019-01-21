package chart

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	"github.com/giantswarm/app-operator/service/controller/app/v1/kubeconfig"
)

func TestResource_GetDesiredState(t *testing.T) {

	tests := []struct {
		name          string
		obj           *v1alpha1.App
		appCatalog    *v1alpha1.AppCatalog
		expectedChart *v1alpha1.Chart
		errorMatcher  func(error) bool
	}{
		{
			name: "case 0: flawless flow",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                        "prometheus",
						"giantswarm.io/cluster":      "6iec4",
						"giantswarm.io/organization": "giantswarm",
						"giantswarm.io/service-type": "managed",
					},
					Annotations: map[string]string{
						"giantswarm.io/managed-by":     "app-operator",
						"giantswarm.io/version-bundle": "0.1.0",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "giantswarm",
					Release: "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "giant-swarm-secret",
							Namespace: "giantswarm",
						},
					},
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
					Name:      "kubernetes-prometheus",
					Namespace: "monitoring",
				},
			},
			appCatalog: &v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.AppCatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					CatalogStorage: v1alpha1.AppCatalogSpecCatalogStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.com/app-catalog/",
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubernetes-prometheus",
					Labels: map[string]string{
						"app":                        "prometheus",
						"giantswarm.io/cluster":      "6iec4",
						"giantswarm.io/organization": "giantswarm",
						"giantswarm.io/service-type": "managed",
					},
					Annotations: map[string]string{
						"giantswarm.io/managed-by":     "app-operator",
						"giantswarm.io/version-bundle": "0.1.0",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:            "giant-swarm-config",
							Namespace:       "giantswarm",
							ResourceVersion: "",
						},
						Secret: v1alpha1.ChartSpecConfigSecret{
							Name:            "giant-swarm-secret",
							Namespace:       "giantswarm",
							ResourceVersion: "",
						},
					},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
		},
		{
			name: "case 1: appcatalog not found",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                        "prometheus",
						"giantswarm.io/cluster":      "6iec4",
						"giantswarm.io/organization": "giantswarm",
						"giantswarm.io/service-type": "managed",
					},
					Annotations: map[string]string{
						"giantswarm.io/managed-by":     "app-operator",
						"giantswarm.io/version-bundle": "0.1.0",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "giantswarm",
					Release: "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "giant-swarm-secret",
							Namespace: "giantswarm",
						},
					},
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
					Name:      "kubernetes-prometheus",
					Namespace: "monitoring",
				},
			},
			appCatalog: &v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm-xxx1",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.AppCatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					CatalogStorage: v1alpha1.AppCatalogSpecCatalogStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.com/app-catalog/",
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			errorMatcher: IsNotFound,
		},
		{
			name: "case 2: generating catalog url failed",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                        "prometheus",
						"giantswarm.io/cluster":      "6iec4",
						"giantswarm.io/organization": "giantswarm",
						"giantswarm.io/service-type": "managed",
					},
					Annotations: map[string]string{
						"giantswarm.io/managed-by":     "app-operator",
						"giantswarm.io/version-bundle": "0.1.0",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "giantswarm",
					Release: "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "giant-swarm-secret",
							Namespace: "giantswarm",
						},
					},
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
					Name:      "kubernetes-prometheus",
					Namespace: "monitoring",
				},
			},
			appCatalog: &v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.AppCatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					CatalogStorage: v1alpha1.AppCatalogSpecCatalogStorage{
						Type: "helm",
						URL:  "", // Empty baseURL
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			errorMatcher: IsFailedExecution,
		},
	}

	var err error
	var kc *kubeconfig.KubeConfig
	{
		c := kubeconfig.Config{
			G8sClient: fake.NewSimpleClientset(),
			K8sClient: k8sfake.NewSimpleClientset(),
			Logger:    microloggertest.New(),
		}

		kc, err = kubeconfig.New(c)
		if err != nil {
			t.Fatalf("error == %#v, want nil", err)
		}
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0, 0)
			if tc.appCatalog != nil {
				objs = append(objs, tc.appCatalog)
			}

			c := Config{
				G8sClient:      fake.NewSimpleClientset(objs...),
				K8sClient:      k8sfake.NewSimpleClientset(),
				KubeConfig:     kc,
				Logger:         microloggertest.New(),
				WatchNamespace: "default",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			result, err := r.GetDesiredState(context.TODO(), tc.obj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if err == nil && tc.errorMatcher == nil {
				chart, err := key.ToChart(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(chart, *tc.expectedChart) {
					t.Fatalf("Chart == %#v, want %#v", chart, tc.expectedChart)
				}
			}
		})
	}
}
