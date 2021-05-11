package configmap

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/values"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/v4/service/controller/app/controllercontext"
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
					Annotations: map[string]string{
						annotation.Notes: "DO NOT EDIT. Values managed by app-operator.",
					},
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
		},
	}

	var err error

	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			objs := make([]runtime.Object, 0)
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

			var valuesService *values.Values
			{
				c := values.Config{
					K8sClient: clientgofake.NewSimpleClientset(objs...),
					Logger:    microloggertest.New(),
				}

				valuesService, err = values.New(c)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}
			}

			c := Config{
				Logger: microloggertest.New(),
				Values: valuesService,

				ChartNamespace: "giantswarm",
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
