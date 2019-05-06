package configmap

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
	clientgofake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func Test_Resource_GetDesiredState(t *testing.T) {
	tests := []struct {
		name              string
		obj               *v1alpha1.App
		appCatalog        v1alpha1.AppCatalog
		configMaps        []*corev1.ConfigMap
		expectedConfigMap *corev1.ConfigMap
		errorMatcher      func(error) bool
	}{
		{
			name: "case 0: configmap is nil when there is no config",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "app-catalog",
					Name:      "test-app",
					Namespace: "kube-system",
				},
			},
			appCatalog: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
			},
			expectedConfigMap: nil,
		},
		{
			name: "case 1: basic match with app config",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-prometheus",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "app-catalog",
					Name:      "prometheus",
					Namespace: "monitoring",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "test-cluster-values",
							Namespace: "giantswarm",
						},
					},
				},
			},
			appCatalog: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
			},
			configMaps: []*corev1.ConfigMap{
				{
					Data: map[string]string{
						"values": "cluster: yaml\n",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-values",
						Namespace: "giantswarm",
					},
				},
			},
			expectedConfigMap: &corev1.ConfigMap{
				Data: map[string]string{
					"values": "cluster: yaml\n",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-prometheus-chart-values",
					Namespace: "giantswarm",
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
		},
		{
			name: "case 2: basic match with catalog config",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "test-catalog",
					Name:      "test-app",
					Namespace: "giantswarm",
				},
			},
			appCatalog: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
				Spec: v1alpha1.AppCatalogSpec{
					Title: "test-catalog",
					Config: v1alpha1.AppCatalogSpecConfig{
						ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
							Name:      "test-catalog-values",
							Namespace: "giantswarm",
						},
					},
				},
			},
			configMaps: []*corev1.ConfigMap{
				{
					Data: map[string]string{
						"values": "catalog: yaml\n",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-catalog-values",
						Namespace: "giantswarm",
					},
				},
			},
			expectedConfigMap: &corev1.ConfigMap{
				Data: map[string]string{
					"values": "catalog: yaml\n",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app-chart-values",
					Namespace: "giantswarm",
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
		},
		{
			name: "case 3: non-intersecting catalog and app config are merged",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Name:      "test-app",
					Namespace: "giantswarm",
					Catalog:   "test-catalog",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "test-cluster-values",
							Namespace: "giantswarm",
						},
					},
				},
			},
			appCatalog: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
				Spec: v1alpha1.AppCatalogSpec{
					Title: "test-catalog",
					Config: v1alpha1.AppCatalogSpecConfig{
						ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
							Name:      "test-catalog-values",
							Namespace: "giantswarm",
						},
					},
				},
			},
			configMaps: []*corev1.ConfigMap{
				{
					Data: map[string]string{
						"values": "catalog: yaml\n",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-catalog-values",
						Namespace: "giantswarm",
					},
				},
				{
					Data: map[string]string{
						"values": "cluster: yaml\n",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-values",
						Namespace: "giantswarm",
					},
				},
			},
			expectedConfigMap: &corev1.ConfigMap{
				Data: map[string]string{
					"values": "catalog: yaml\ncluster: yaml\n",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app-chart-values",
					Namespace: "giantswarm",
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
		},
		{
			name: "case 4: intersecting catalog and app config are merged, app is preferred",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Name:      "test-app",
					Namespace: "giantswarm",
					Catalog:   "test-catalog",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "test-cluster-values",
							Namespace: "giantswarm",
						},
					},
				},
			},
			appCatalog: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
				Spec: v1alpha1.AppCatalogSpec{
					Title: "test-catalog",
					Config: v1alpha1.AppCatalogSpecConfig{
						ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
							Name:      "test-catalog-values",
							Namespace: "giantswarm",
						},
					},
				},
			},
			configMaps: []*corev1.ConfigMap{
				{
					Data: map[string]string{
						"values": "test: catalog\n",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-catalog-values",
						Namespace: "giantswarm",
					},
				},
				{
					Data: map[string]string{
						"values": "test: app\n",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-values",
						Namespace: "giantswarm",
					},
				},
			},
			expectedConfigMap: &corev1.ConfigMap{
				Data: map[string]string{
					"values": "test: app\n",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app-chart-values",
					Namespace: "giantswarm",
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
		},
		{
			name: "case 5: intersecting catalog, app and user config is merged, user is preferred",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Name:      "test-app",
					Namespace: "giantswarm",
					Catalog:   "test-catalog",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "test-cluster-values",
							Namespace: "giantswarm",
						},
					},
					UserConfig: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "test-user-values",
							Namespace: "giantswarm",
						},
					},
				},
			},
			appCatalog: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
				Spec: v1alpha1.AppCatalogSpec{
					Title: "test-catalog",
					Config: v1alpha1.AppCatalogSpecConfig{
						ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
							Name:      "test-catalog-values",
							Namespace: "giantswarm",
						},
					},
				},
			},
			configMaps: []*corev1.ConfigMap{
				{
					Data: map[string]string{
						"values": "catalog: test\ntest: catalog\n",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-catalog-values",
						Namespace: "giantswarm",
					},
				},
				{
					Data: map[string]string{
						"values": "cluster: test\ntest: app\n",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-values",
						Namespace: "giantswarm",
					},
				},
				{
					Data: map[string]string{
						"values": "user: test\ntest: user\n",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-user-values",
						Namespace: "giantswarm",
					},
				},
			},
			expectedConfigMap: &corev1.ConfigMap{
				Data: map[string]string{
					"values": "catalog: test\ncluster: test\ntest: user\nuser: test\n",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app-chart-values",
					Namespace: "giantswarm",
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0, 0)
			for _, cm := range tc.configMaps {
				objs = append(objs, cm)
			}

			var ctx context.Context
			{
				c := controllercontext.Context{
					AppCatalog: tc.appCatalog,
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			c := Config{
				G8sClient: fake.NewSimpleClientset(),
				K8sClient: clientgofake.NewSimpleClientset(objs...),
				Logger:    microloggertest.New(),

				ChartNamespace: "giantswarm",
				ProjectName:    "app-operator",
				WatchNamespace: "default",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
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

			if result != nil && tc.expectedConfigMap == nil {
				t.Fatalf("expected nil configmap got %#v", result)
			}
			if result == nil && tc.expectedConfigMap != nil {
				t.Fatal("expected non-nil configmap got nil")
			}

			if tc.expectedConfigMap != nil {
				configMap, err := toConfigMap(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(configMap.ObjectMeta, tc.expectedConfigMap.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(configMap.ObjectMeta, tc.expectedConfigMap.ObjectMeta))
				}
				if !reflect.DeepEqual(configMap.Data, tc.expectedConfigMap.Data) {
					t.Fatalf("want matching data \n %s", cmp.Diff(configMap.Data, tc.expectedConfigMap.Data))
				}
			}
		})
	}
}
