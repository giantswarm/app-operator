package configmap

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/values"
)

func Test_Resource_GetCurrentState(t *testing.T) {
	testCases := []struct {
		name              string
		obj               *v1alpha1.App
		configMap         *corev1.ConfigMap
		expectedConfigMap *corev1.ConfigMap
		errorMatcher      func(error) bool
	}{
		{
			name: "case 0: basic match",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "app-values",
							Namespace: "default",
						},
					},
					Namespace: "kube-system",
				},
			},
			configMap: &corev1.ConfigMap{
				Data: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-chart-values",
					Namespace: "giantswarm",
				},
			},
			expectedConfigMap: &corev1.ConfigMap{
				Data: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-chart-values",
					Namespace: "giantswarm",
				},
			},
		},
		{
			name: "case 1: no matching configmap",
			obj: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "app-values",
							Namespace: "default",
						},
					},
					Namespace: "kube-system",
				},
			},
			configMap: &corev1.ConfigMap{
				Data: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-app-values",
					Namespace: "default",
				},
			},
			expectedConfigMap: &corev1.ConfigMap{},
		},
		{
			name: "case 2: namespace does not match",
			obj: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "app-values",
							Namespace: "default",
						},
					},
					Namespace: "kube-system",
				},
			},
			configMap: &corev1.ConfigMap{
				Data: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
			expectedConfigMap: &corev1.ConfigMap{},
		},
		{
			name: "case 3: no configmaps",
			obj: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "app-values",
							Namespace: "default",
						},
					},
				},
			},
			expectedConfigMap: &corev1.ConfigMap{},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			objs := make([]runtime.Object, 0, 0)
			if tc.configMap != nil {
				objs = append(objs, tc.configMap)
			}

			k8sClient := clientgofake.NewSimpleClientset(objs...)

			var err error

			var ctx context.Context
			{
				c := controllercontext.Context{
					K8sClient: k8sClient,
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			var valuesService *values.Values
			{
				c := values.Config{
					K8sClient: clientgofake.NewSimpleClientset(),
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
				ProjectName:    "app-operator",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			result, err := r.GetCurrentState(ctx, tc.obj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			configMap, err := toConfigMap(result)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if !reflect.DeepEqual(configMap, tc.expectedConfigMap) {
				t.Fatalf("want matching configmap \n %s", cmp.Diff(configMap, tc.expectedConfigMap))
			}
		})
	}
}
