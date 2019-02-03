package secret

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/micrologger/microloggertest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func Test_Resource_GetCurrentState(t *testing.T) {
	tests := []struct {
		name           string
		obj            *v1alpha1.App
		secret         *corev1.Secret
		expectedSecret *corev1.Secret
		errorMatcher   func(error) bool
	}{
		{
			name: "case 0: basic match",
			obj: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "app-secrets",
							Namespace: "default",
						},
					},
				},
			},
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
				},
			},
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
				},
			},
		},
		{
			name: "case 1: no matching secret",
			obj: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "app-secrets",
							Namespace: "default",
						},
					},
				},
			},
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-app-secrets",
					Namespace: "default",
				},
			},
		},
		{
			name: "case 2: no secrets",
			obj: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "app-secrets",
							Namespace: "default",
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
				G8sClient: g8sClient,
				K8sClient: k8sClient,
				Logger:    microloggertest.New(),

				ProjectName:    "app-operator",
				WatchNamespace: "default",
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
				t.Fatalf("secret == %q, want %q", secret, tc.expectedSecret)
			}
		})
	}
}
