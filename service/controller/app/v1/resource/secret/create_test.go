package secret

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
	clientgofake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/service/controller/app/v1/values"
)

func Test_Resource_newCreateChange(t *testing.T) {
	testCases := []struct {
		name           string
		obj            v1alpha1.App
		currentState   *corev1.Secret
		desiredState   *corev1.Secret
		expectedSecret *corev1.Secret
	}{
		{
			name:         "case 0: empty current and non-empty desired, expected desired",
			currentState: &corev1.Secret{},
			desiredState: &corev1.Secret{
				StringData: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
			expectedSecret: &corev1.Secret{
				StringData: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
		},
		{
			name: "case 1: equal current and desired states, expected empty",
			currentState: &corev1.Secret{
				StringData: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
			desiredState: &corev1.Secret{
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
			name: "case 2: non-equal current and desired states, expected empty",
			currentState: &corev1.Secret{
				StringData: map[string]string{
					"key": "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
				},
			},
			desiredState: &corev1.Secret{
				StringData: map[string]string{
					"another": "value",
					"key":     "value",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-values",
					Namespace: "default",
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

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result, err := r.newCreateChange(context.Background(), tc.currentState, tc.desiredState)
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
