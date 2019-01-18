package chart

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/micrologger/microloggertest"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func TestResource_GetDesiredState(t *testing.T) {

	tests := []struct {
		name          string
		appObj        *v1alpha1.App
		appCatalog    *v1alpha1.AppCatalog
		expectedChart *v1alpha1.Chart
		errorMatcher  func(error) bool
	}{
		{
			name: "case 0: flawless flow",
			appObj: &v1alpha1.App{
				ObjectMeta: v1.ObjectMeta{
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
							Name:      "ginat-swarm-config",
							Namespace: "giantswarm",
						},
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "ginat-swarm-secret",
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
				ObjectMeta: v1.ObjectMeta{
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
				TypeMeta: v1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: v1.ObjectMeta{
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
							Name:            "ginat-swarm-config",
							Namespace:       "giantswarm",
							ResourceVersion: "",
						},
						Secret: v1alpha1.ChartSpecConfigSecret{
							Name:            "ginat-swarm-secret",
							Namespace:       "giantswarm",
							ResourceVersion: "",
						},
					},
					Name: "my-cool-prometheus",
					KubeConfig: v1alpha1.ChartSpecKubeConfig{
						Secret: v1alpha1.ChartSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/-kubernetes-prometheus-1.0.0.tgz",
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0, 0)
			if tc.appCatalog != nil {
				objs = append(objs, tc.appCatalog)
			}
			c := Config{
				K8sClient: k8sfake.NewSimpleClientset(),
				G8sClient: fake.NewSimpleClientset(objs...),
				Logger:    microloggertest.New(),
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			result, err := r.GetDesiredState(context.TODO(), tc.appObj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			chart, err := key.ToChart(result)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if !reflect.DeepEqual(chart, *tc.expectedChart) {
				t.Fatalf("Chart == %#v, want %#v", chart, tc.expectedChart)
			}
		})
	}
}
