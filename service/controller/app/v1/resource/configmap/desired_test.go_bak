package configmap

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
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
		/*
			{
				name: "case 0: empty app configmap",
				obj: &v1alpha1.App{
					Spec: v1alpha1.AppSpec{
						Catalog: "test-catalog",
						Config: v1alpha1.AppSpecConfig{
							ConfigMap: v1alpha1.AppSpecConfigConfigMap{
								Name:      "test-app-values",
								Namespace: "default",
							},
						},
						Namespace: "giantswarm",
					},
				},
				appCatalog: v1alpha1.AppCatalog{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-catalog",
						Namespace: "default",
					},
				},
				configMaps: []*corev1.ConfigMap{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
				},
				expectedConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-values",
						Namespace: "giantswarm",
						Labels: map[string]string{
							"giantswarm.io/managed-by": "app-operator",
						},
					},
				},
			},
		*/
		{
			name: "case 1: app configmap with values",
			obj: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Catalog: "test-catalog",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
					Namespace: "giantswarm",
				},
			},
			appCatalog: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-catalog",
					Namespace: "default",
				},
			},
			configMaps: []*corev1.ConfigMap{
				{
					Data: map[string]string{
						"values": "app: value\n"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-values",
						Namespace: "default",
					},
				},
			},
			expectedConfigMap: &corev1.ConfigMap{
				Data: map[string]string{
					"values": "app: value\n",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-values",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"giantswarm.io/managed-by": "app-operator",
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

			if err == nil && tc.errorMatcher == nil {
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

func Test_Resource_GetDesiredState(t *testing.T) {

}
