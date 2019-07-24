package chart

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/pkg/annotation"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func Test_Resource_GetDesiredState(t *testing.T) {
	tests := []struct {
		name          string
		obj           *v1alpha1.App
		appCatalog    v1alpha1.AppCatalog
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
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "prometheus",
					Namespace: "monitoring",
					Version:   "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
				},
			},
			appCatalog: v1alpha1.AppCatalog{
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
					Storage: v1alpha1.AppCatalogSpecStorage{
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
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "my-cool-prometheus-chart-values",
							Namespace: "giantswarm",
						},
					},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/prometheus-1.0.0.tgz",
				},
			},
		},
		{
			name: "case 1: generating catalog url failed",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "kubernetes-prometheus",
					Namespace: "monitoring",
					Version:   "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
				},
			},
			appCatalog: v1alpha1.AppCatalog{
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
					Storage: v1alpha1.AppCatalogSpecStorage{
						Type: "helm",
						URL:  "", // Empty baseURL
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			errorMatcher: IsExecutionFailed,
		},
		{
			name: "case 2: cordoned app",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotation.CordonReason: "migrating job in process",
						annotation.CordonUntil:  "2019-12-31T23:59:59Z",
					},
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "kubernetes-prometheus",
					Namespace: "monitoring",
					Version:   "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
				},
			},
			appCatalog: v1alpha1.AppCatalog{
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
					Storage: v1alpha1.AppCatalogSpecStorage{
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
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/cordon-reason": "migrating job in process",
						"chart-operator.giantswarm.io/cordon-until":  "2019-12-31T23:59:59Z",
					},
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "my-cool-prometheus-chart-values",
							Namespace: "giantswarm",
						},
					},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			ns := make([]runtime.Object, 0, 0)
			if tc.obj != nil {
				ns = append(ns, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tc.obj.Namespace}})
			}
			k8sClient := k8sfake.NewSimpleClientset(ns...)

			c := Config{
				G8sClient: fake.NewSimpleClientset(),
				K8sClient: k8sClient,
				Logger:    microloggertest.New(),

				ChartNamespace: "giantswarm",
				ProjectName:    "app-operator",
				WatchNamespace: "default",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var ctx context.Context
			{
				c := controllercontext.Context{
					AppCatalog: tc.appCatalog,
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			result, err := r.GetDesiredState(ctx, tc.obj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if err == nil && tc.errorMatcher == nil {
				chart, err := toChart(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(chart.ObjectMeta, tc.expectedChart.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(chart.ObjectMeta, tc.expectedChart.ObjectMeta))
				}

				if !reflect.DeepEqual(chart.Spec, tc.expectedChart.Spec) {
					t.Fatalf("want matching spec \n %s", cmp.Diff(chart.Spec, tc.expectedChart.Spec))
				}

				if !reflect.DeepEqual(chart.TypeMeta, tc.expectedChart.TypeMeta) {
					t.Fatalf("want matching typemeta \n %s", cmp.Diff(chart.TypeMeta, tc.expectedChart.TypeMeta))
				}
			}
		})
	}
}

func Test_generateConfig(t *testing.T) {
	tests := []struct {
		name           string
		cr             v1alpha1.App
		appCatalog     v1alpha1.AppCatalog
		expectedConfig v1alpha1.ChartSpecConfig
	}{
		{
			name:           "case 0: no config",
			cr:             v1alpha1.App{},
			appCatalog:     v1alpha1.AppCatalog{},
			expectedConfig: v1alpha1.ChartSpecConfig{},
		},
		{
			name: "case 1: has a configmap from app",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
					Namespace: "giantswarm",
				},
			},
			appCatalog: v1alpha1.AppCatalog{},
			expectedConfig: v1alpha1.ChartSpecConfig{
				ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
					Name:      "test-app-chart-values",
					Namespace: "giantswarm",
				},
			},
		},
		{
			name: "case 2: has a secret from app",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
					Namespace: "giantswarm",
				},
			},
			appCatalog: v1alpha1.AppCatalog{},
			expectedConfig: v1alpha1.ChartSpecConfig{
				Secret: v1alpha1.ChartSpecConfigSecret{
					Name:      "test-app-chart-secrets",
					Namespace: "giantswarm",
				},
			},
		},
		{
			name: "case 3: has both a configmap and secret from app",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "test-app-values",
							Namespace: "default",
						},
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
					Namespace: "giantswarm",
				},
			},
			appCatalog: v1alpha1.AppCatalog{},
			expectedConfig: v1alpha1.ChartSpecConfig{
				ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
					Name:      "test-app-chart-values",
					Namespace: "giantswarm",
				},
				Secret: v1alpha1.ChartSpecConfigSecret{
					Name:      "test-app-chart-secrets",
					Namespace: "giantswarm",
				},
			},
		},
		{
			name: "case 4: has a configmap from catalog",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			appCatalog: v1alpha1.AppCatalog{
				Spec: v1alpha1.AppCatalogSpec{
					Config: v1alpha1.AppCatalogSpecConfig{
						ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
				},
			},
			expectedConfig: v1alpha1.ChartSpecConfig{
				ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
					Name:      "test-app-chart-values",
					Namespace: "giantswarm",
				},
			},
		},
		{
			name: "case 5: has a secret from catalog",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			appCatalog: v1alpha1.AppCatalog{
				Spec: v1alpha1.AppCatalogSpec{
					Config: v1alpha1.AppCatalogSpecConfig{
						Secret: v1alpha1.AppCatalogSpecConfigSecret{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
				},
			},
			expectedConfig: v1alpha1.ChartSpecConfig{
				Secret: v1alpha1.ChartSpecConfigSecret{
					Name:      "test-app-chart-secrets",
					Namespace: "giantswarm",
				},
			},
		},
		{
			name: "case 6: has both a configmap and secret from catalog",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			appCatalog: v1alpha1.AppCatalog{
				Spec: v1alpha1.AppCatalogSpec{
					Config: v1alpha1.AppCatalogSpecConfig{
						ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
							Name:      "test-app-values",
							Namespace: "default",
						},
						Secret: v1alpha1.AppCatalogSpecConfigSecret{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
				},
			},
			expectedConfig: v1alpha1.ChartSpecConfig{
				ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
					Name:      "test-app-chart-values",
					Namespace: "giantswarm",
				},
				Secret: v1alpha1.ChartSpecConfigSecret{
					Name:      "test-app-chart-secrets",
					Namespace: "giantswarm",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := generateConfig(tc.cr, tc.appCatalog, "giantswarm")

			if !reflect.DeepEqual(result, tc.expectedConfig) {
				t.Fatalf("want matching Config \n %s", cmp.Diff(result, tc.expectedConfig))
			}
		})
	}
}

func Test_processLabels(t *testing.T) {
	tests := []struct {
		name           string
		projectName    string
		inputLabels    map[string]string
		expectedLabels map[string]string
	}{
		{
			name:        "case 0: basic match",
			projectName: "app-operator",
			inputLabels: map[string]string{
				"app-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":           "release-operator",
			},
			expectedLabels: map[string]string{
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":             "app-operator",
			},
		},
		{
			name:        "case 1: extra labels still present",
			projectName: "app-operator",
			inputLabels: map[string]string{
				"app":                                "prometheus",
				"app-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/cluster":              "5xchu",
				"giantswarm.io/managed-by":           "cluster-operator",
				"giantswarm.io/organization":         "giantswarm",
			},
			expectedLabels: map[string]string{
				"app":                                  "prometheus",
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/cluster":                "5xchu",
				"giantswarm.io/managed-by":             "app-operator",
				"giantswarm.io/organization":           "giantswarm",
			},
		},
		{
			name:        "case 2: empty inputs",
			projectName: "app-operator",
			expectedLabels: map[string]string{
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":             "app-operator",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			result := processLabels(tc.projectName, tc.inputLabels)

			if !reflect.DeepEqual(result, tc.expectedLabels) {
				t.Fatalf("want matching \n %s", cmp.Diff(result, tc.expectedLabels))
			}
		})
	}
}

func Test_processAnnotations(t *testing.T) {
	tests := []struct {
		name                string
		obj                 *v1alpha1.App
		expectedAnnotations map[string]string
	}{
		{
			name: "case 0: basic match",
			obj: &v1alpha1.App{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"app-operator.giantswarm.io/cordon-reason": "migrating job in process",
						"app-operator.giantswarm.io/cordon-until":  "2019-12-31T23:59:59Z",
					},
				},
			},
			expectedAnnotations: map[string]string{
				"chart-operator.giantswarm.io/cordon-reason": "migrating job in process",
				"chart-operator.giantswarm.io/cordon-until":  "2019-12-31T23:59:59Z",
			},
		},
		{
			name: "case 1: filter other annotations",
			obj: &v1alpha1.App{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubectl.kubernetes.io/last-applied-configuration": "application.giantswarm.io/v1alpha1",
					},
				},
			},
			expectedAnnotations: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			result := processAnnotations(*tc.obj)

			if !reflect.DeepEqual(result, tc.expectedAnnotations) {
				t.Fatalf("want matching \n %s", cmp.Diff(result, tc.expectedAnnotations))
			}
		})
	}
}
