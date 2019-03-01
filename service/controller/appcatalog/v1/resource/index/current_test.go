package index

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/service/controller/appcatalog/v1/key"
)

func Test_Resource_GetCurrentState(t *testing.T) {
	tests := []struct {
		name              string
		obj               interface{}
		expectedConfigMap *corev1.ConfigMap
		errorMatcher      func(error) bool
	}{
		{
			name: "case 0: index configMap already created",
			obj: &v1alpha1.AppCatalog{
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
			expectedConfigMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm-index",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"giantswarm.io/managed-by": "app-operator",
					},
				},
				Data: map[string]string{
					"index.yaml": "test yaml",
				},
			},
		},
		{
			name: "case 1: index configMap not found",
			obj: &v1alpha1.AppCatalog{
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
			expectedConfigMap: nil,
		},
		{
			name:         "case 2: wrong obj type",
			obj:          &v1alpha1.App{},
			errorMatcher: key.IsWrongTypeError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0, 0)
			if tc.expectedConfigMap != nil {
				objs = append(objs, tc.expectedConfigMap)
			}

			k8sClient := fake.NewSimpleClientset(objs...)

			c := Config{
				K8sClient: k8sClient,
				Logger:    microloggertest.New(),

				IndexNamespace: "giantswarm",
				ProjectName:    "app-operator",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			result, err := r.GetCurrentState(context.Background(), tc.obj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if result != nil && tc.expectedConfigMap == nil {
				t.Fatalf("expected nil ConfigMap got %#v", result)
			}
			if result == nil && tc.expectedConfigMap != nil {
				t.Fatal("expected non-nil ConfigMap got nil")
			}

			if len(result) == 1 {
				cm, err := toConfigMap(result[0])
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if tc.expectedConfigMap != nil && !reflect.DeepEqual(cm, tc.expectedConfigMap) {
					t.Fatalf("ConfigMap == %#v, want %#v", cm, tc.expectedConfigMap)
				}
			}
		})
	}
}
