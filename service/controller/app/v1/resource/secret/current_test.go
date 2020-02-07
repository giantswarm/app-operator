package secret

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
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
		name           string
		obj            *v1alpha1.App
		secret         *corev1.Secret
		expectedSecret *corev1.Secret
		errorMatcher   func(error) bool
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
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "app-secrets",
							Namespace: "default",
						},
					},
					Namespace: "kube-system",
				},
			},
			secret: &corev1.Secret{
				StringData: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-chart-secrets",
					Namespace: "giantswarm",
				},
			},
			expectedSecret: &corev1.Secret{
				StringData: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-chart-secrets",
					Namespace: "giantswarm",
				},
			},
		},
		{
			name: "case 1: no matching secret",
			obj: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "app-values",
							Namespace: "default",
						},
					},
					Namespace: "kube-system",
				},
			},
			secret: &corev1.Secret{
				StringData: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-app-values",
					Namespace: "default",
				},
			},
			expectedSecret: &corev1.Secret{},
		},
		{
			name: "case 2: namespace does not match",
			obj: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "app-values",
							Namespace: "default",
						},
					},
					Namespace: "kube-system",
				},
			},
			secret: &corev1.Secret{
				StringData: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
			expectedSecret: &corev1.Secret{},
		},
		{
			name: "case 3: no secrets",
			obj: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "app-values",
							Namespace: "default",
						},
					},
				},
			},
			expectedSecret: &corev1.Secret{},
		},
	}

	var err error

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

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			objs := make([]runtime.Object, 0, 0)
			if tc.secret != nil {
				objs = append(objs, tc.secret)
			}

			g8sClient := fake.NewSimpleClientset()
			k8sClient := clientgofake.NewSimpleClientset(objs...)

			var ctx context.Context
			{
				c := controllercontext.Context{
					G8sClient: g8sClient,
					K8sClient: k8sClient,
				}
				ctx = controllercontext.NewContext(context.Background(), c)
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

			result, err := r.GetCurrentState(ctx, tc.obj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			secret, err := toSecret(result)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if !reflect.DeepEqual(secret, tc.expectedSecret) {
				t.Fatalf("want matching secret \n %s", cmp.Diff(secret, tc.expectedSecret))
			}
		})
	}
}
