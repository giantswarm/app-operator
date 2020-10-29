package values

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
)

func Test_MergeConfigMapData(t *testing.T) {
	tests := []struct {
		name         string
		app          v1alpha1.App
		appCatalog   v1alpha1.AppCatalog
		configMaps   []*corev1.ConfigMap
		expectedData map[string]string
		errorMatcher func(error) bool
	}{
		{
			name: "case 0: configmap is nil when there is no config",
			app: v1alpha1.App{
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
			expectedData: nil,
		},
		{
			name: "case 1: basic match with app config",
			app: v1alpha1.App{
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
			expectedData: map[string]string{
				"values": "cluster: yaml\n",
			},
		},
		{
			name: "case 2: basic match with catalog config",
			app: v1alpha1.App{
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
			expectedData: map[string]string{
				"values": "catalog: yaml\n",
			},
		},
		{
			name: "case 3: non-intersecting catalog and app config are merged",
			app: v1alpha1.App{
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
			expectedData: map[string]string{
				"values": "catalog: yaml\ncluster: yaml\n",
			},
		},
		{
			name: "case 4: intersecting catalog and app config are merged, app is preferred",
			app: v1alpha1.App{
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
			expectedData: map[string]string{
				"values": "test: app\n",
			},
		},
		{
			name: "case 5: intersecting catalog, app and user config is merged, user is preferred",
			app: v1alpha1.App{
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
					UserConfig: v1alpha1.AppSpecUserConfig{
						ConfigMap: v1alpha1.AppSpecUserConfigConfigMap{
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
			expectedData: map[string]string{
				"values": "catalog: test\ncluster: test\ntest: user\nuser: test\n",
			},
		},
		{
			name: "case 6: parsing error from wrong user values",
			app: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "test-catalog",
					Name:      "test-app",
					Namespace: "giantswarm",
					UserConfig: v1alpha1.AppSpecUserConfig{
						ConfigMap: v1alpha1.AppSpecUserConfigConfigMap{
							Name:      "user-values",
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
						"values": `values: val`,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-catalog-values",
						Namespace: "giantswarm",
					},
				},
				{
					Data: map[string]string{
						"values": `values: -`,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "user-values",
						Namespace: "giantswarm",
					},
				},
			},
			errorMatcher: IsParsingError,
		},
	}

	ctx := context.Background()

	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			objs := make([]runtime.Object, 0)
			for _, cm := range tc.configMaps {
				objs = append(objs, cm)
			}

			c := Config{
				K8sClient: clientgofake.NewSimpleClientset(objs...),
				Logger:    microloggertest.New(),
			}
			v, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			result, err := v.MergeConfigMapData(ctx, tc.app, tc.appCatalog)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if result != nil && tc.expectedData == nil {
				t.Fatalf("expected nil map got %#v", result)
			}
			if result == nil && tc.expectedData != nil {
				t.Fatal("expected non-nil gmap got nil")
			}

			if tc.expectedData != nil {
				if !reflect.DeepEqual(result, tc.expectedData) {
					t.Fatalf("want matching data \n %s", cmp.Diff(result, tc.expectedData))
				}
			}
		})
	}
}
