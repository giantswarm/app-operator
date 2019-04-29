package secret

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
		name           string
		obj            *v1alpha1.App
		appCatalog     v1alpha1.AppCatalog
		secrets        []*corev1.Secret
		expectedSecret *corev1.Secret
		errorMatcher   func(error) bool
	}{
		{
			name: "case 0: secret is nil when there are no secrets",
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
			expectedSecret: nil,
		},
		{
			name: "case 1: basic match with app secrets",
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
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"values": []byte("cluster: yaml\n"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-prometheus-chart-secrets",
					Namespace: "monitoring",
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
		},
		{
			name: "case 2: basic match with catalog secrets",
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
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"values": []byte("catalog: yaml\n"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app-chart-secrets",
					Namespace: "giantswarm",
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
		},
		{
			name: "case 3: non-intersecting catalog and app secrets are merged",
			obj: &v1alpha1.App{
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
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"values": []byte("catalog: yaml\ncluster: yaml\n"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app-chart-secrets",
					Namespace: "giantswarm",
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
		},
		{
			name: "case 4: intersecting catalog and app secrets, app overwrites catalog",
			obj: &v1alpha1.App{
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
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"values": []byte("catalog: yaml\ncluster: yaml\ntest: app\n"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app-chart-secrets",
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
			for _, cm := range tc.secrets {
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

			if result != nil && tc.expectedSecret == nil {
				t.Fatalf("expected nil secret got %#v", result)
			}
			if result == nil && tc.expectedSecret != nil {
				t.Fatal("expected non-nil secret got nil")
			}

			if tc.expectedSecret != nil {
				secret, err := toSecret(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(secret.ObjectMeta, tc.expectedSecret.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(secret.ObjectMeta, tc.expectedSecret.ObjectMeta))
				}
				if !reflect.DeepEqual(secret.Data, tc.expectedSecret.Data) {
					data := toStringMap(secret.Data)
					expectedData := toStringMap(tc.expectedSecret.Data)

					t.Fatalf("want matching data \n %s", cmp.Diff(data, expectedData))
				}
			}
		})
	}
}

func toStringMap(input map[string][]byte) map[string]string {
	result := map[string]string{}

	for k, v := range input {
		result[k] = string(v)
	}

	return result
}
