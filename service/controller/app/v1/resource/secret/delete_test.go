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
	clientgofake "k8s.io/client-go/kubernetes/fake"
)

func Test_Resource_newDeleteChange(t *testing.T) {
	testCases := []struct {
		name           string
		obj            v1alpha1.App
		currentState   *corev1.Secret
		desiredState   *corev1.Secret
		expectedSecret *corev1.Secret
	}{
		{
			name:           "case 0: empty current and desired, expected empty",
			currentState:   &corev1.Secret{},
			desiredState:   &corev1.Secret{},
			expectedSecret: &corev1.Secret{},
		},
		{
			name: "case 1: non empty current and desired, expected desired",
			currentState: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte("value"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
			desiredState: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte("value"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte("value"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
		},
		{
			name: "case 2: different current and desired, expected empty",
			currentState: &corev1.Secret{
				Data: map[string][]byte{
					"another": []byte("value"),
					"key":     []byte("value"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
			desiredState: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte("value"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
			expectedSecret: &corev1.Secret{},
		},
	}

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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := r.newDeleteChange(context.Background(), tc.obj, tc.currentState, tc.desiredState)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
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
