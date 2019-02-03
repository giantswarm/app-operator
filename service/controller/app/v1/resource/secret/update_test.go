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
	clientgofake "k8s.io/client-go/kubernetes/fake"
)

func Test_Resource_newUpdateChange(t *testing.T) {
	testCases := []struct {
		name           string
		obj            v1alpha1.App
		currentState   *corev1.Secret
		desiredState   *corev1.Secret
		expectedSecret *corev1.Secret
	}{
		{
			name:         "case 0: empty current and non-empty desired, expected empty",
			currentState: &corev1.Secret{},
			desiredState: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
				},
			},
			expectedSecret: &corev1.Secret{},
		},
		{
			name: "case 1: equal current and desired states, expected empty",
			currentState: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
				},
			},
			desiredState: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
				},
			},
			expectedSecret: &corev1.Secret{},
		},
		{
			name: "case 2: non-equal data, expected desired",
			currentState: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
				},
			},
			desiredState: &corev1.Secret{
				Data: map[string][]byte{
					"another": []byte{},
					"key":     []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
				},
			},
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"another": []byte{},
					"key":     []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
				},
			},
		},
		{
			name: "case 3: non-equal metadata, expected desired",
			currentState: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
				},
			},
			desiredState: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
					Labels: map[string]string{
						"giantswarm.io/cluster": "5xchu",
					},
				},
			},
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-secrets",
					Namespace: "default",
					Labels: map[string]string{
						"giantswarm.io/cluster": "5xchu",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := Config{
				G8sClient: fake.NewSimpleClientset(),
				K8sClient: clientgofake.NewSimpleClientset(),
				Logger:    microloggertest.New(),

				ProjectName:    "app-operator",
				WatchNamespace: "default",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			result, err := r.newUpdateChange(context.Background(), tc.currentState, tc.desiredState)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			configMap, err := toSecret(result)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if !reflect.DeepEqual(configMap, tc.expectedSecret) {
				t.Fatalf("configMap == %q, want %q", configMap, tc.expectedSecret)
			}
		})
	}
}
