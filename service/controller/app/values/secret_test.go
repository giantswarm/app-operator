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

func Test_MergeSecretData(t *testing.T) {
	tests := []struct {
		name         string
		app          v1alpha1.App
		appCatalog   v1alpha1.AppCatalog
		secrets      []*corev1.Secret
		expectedData map[string][]byte
		errorMatcher func(error) bool
	}{
		{
			name: "case 0: secret is nil when there are no secrets",
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
			name: "case 1: basic match with app secrets",
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
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test-cluster-secrets",
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
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"secrets": []byte("cluster: yaml\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-secrets",
						Namespace: "giantswarm",
					},
				},
			},
			expectedData: map[string][]byte{
				"values": []byte("cluster: yaml\n"),
			},
		},
		{
			name: "case 2: basic match with catalog secrets",
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
						Secret: v1alpha1.AppCatalogSpecConfigSecret{
							Name:      "test-catalog-secrets",
							Namespace: "giantswarm",
						},
					},
				},
			},
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"secrets": []byte("catalog: yaml\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-catalog-secrets",
						Namespace: "giantswarm",
					},
				},
			},
			expectedData: map[string][]byte{
				"values": []byte("catalog: yaml\n"),
			},
		},
		{
			name: "case 3: non-intersecting catalog and app secrets are merged",
			app: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "test-catalog",
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test-cluster-secrets",
							Namespace: "giantswarm",
						},
					},
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
						Secret: v1alpha1.AppCatalogSpecConfigSecret{
							Name:      "test-catalog-secrets",
							Namespace: "giantswarm",
						},
					},
				},
			},
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"values": []byte("catalog: yaml\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-catalog-secrets",
						Namespace: "giantswarm",
					},
				},
				{
					Data: map[string][]byte{
						"values": []byte("cluster: yaml\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-secrets",
						Namespace: "giantswarm",
					},
				},
			},
			expectedData: map[string][]byte{
				"values": []byte("catalog: yaml\ncluster: yaml\n"),
			},
		},
		{
			name: "case 4: intersecting catalog and app secrets, app overwrites catalog",
			app: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "test-catalog",
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test-cluster-secrets",
							Namespace: "giantswarm",
						},
					},
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
						Secret: v1alpha1.AppCatalogSpecConfigSecret{
							Name:      "test-catalog-secrets",
							Namespace: "giantswarm",
						},
					},
				},
			},
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"values": []byte("catalog: yaml\ntest: catalog\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-catalog-secrets",
						Namespace: "giantswarm",
					},
				},
				{
					Data: map[string][]byte{
						"values": []byte("cluster: yaml\ntest: app\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-secrets",
						Namespace: "giantswarm",
					},
				},
			},
			expectedData: map[string][]byte{
				// "test: app" overrides "test: catalog".
				"values": []byte("catalog: yaml\ncluster: yaml\ntest: app\n"),
			},
		},
		{
			name: "case 5: intersecting catalog, app and user secrets are merged, user is preferred",
			app: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "test-catalog",
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test-cluster-secrets",
							Namespace: "giantswarm",
						},
					},
					Name:      "test-app",
					Namespace: "giantswarm",
					UserConfig: v1alpha1.AppSpecUserConfig{
						Secret: v1alpha1.AppSpecUserConfigSecret{
							Name:      "test-user-secrets",
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
						Secret: v1alpha1.AppCatalogSpecConfigSecret{
							Name:      "test-catalog-secrets",
							Namespace: "giantswarm",
						},
					},
				},
			},
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"values": []byte("catalog: test\ntest: catalog\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-catalog-secrets",
						Namespace: "giantswarm",
					},
				},
				{
					Data: map[string][]byte{
						"values": []byte("cluster: test\ntest: app\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-secrets",
						Namespace: "giantswarm",
					},
				},
				{
					Data: map[string][]byte{
						"values": []byte("user: test\ntest: user\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-user-secrets",
						Namespace: "giantswarm",
					},
				},
			},
			expectedData: map[string][]byte{
				// "test: user" overrides "test: catalog" and "test: app".
				"values": []byte("catalog: test\ncluster: test\ntest: user\nuser: test\n"),
			},
		},
		{
			name: "case 6: parsing error from wrong user values",
			app: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-prometheus",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "app-catalog",
					Name:      "prometheus",
					Namespace: "monitoring",
					UserConfig: v1alpha1.AppSpecUserConfig{
						Secret: v1alpha1.AppSpecUserConfigSecret{
							Name:      "test-cluster-user-secrets",
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
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"secrets": []byte("cluster --\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-user-secrets",
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
			for _, cm := range tc.secrets {
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

			result, err := v.MergeSecretData(ctx, tc.app, tc.appCatalog)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if result != nil && tc.expectedData == nil {
				t.Fatalf("expected nil secret got %#v", result)
			}
			if result == nil && tc.expectedData != nil {
				t.Fatal("expected non-nil secret got nil")
			}

			if tc.expectedData != nil {
				if !reflect.DeepEqual(result, tc.expectedData) {
					data := toStringMap(result)
					expectedData := toStringMap(tc.expectedData)

					t.Fatalf("want matching data \n %s", cmp.Diff(data, expectedData))
				}
			}
		})
	}
}
