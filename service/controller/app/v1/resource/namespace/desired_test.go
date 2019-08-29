package namespace

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

	"github.com/giantswarm/app-operator/pkg/label"
)

func Test_Resource_Namespace_GetDesiredState(t *testing.T) {
	testCases := []struct {
		name           string
		obj            interface{}
		configMap      corev1.ConfigMap
		expectedName   string
		expectedLabels map[string]string
	}{
		{
			name: "case 0: basic match",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testapp",
					Namespace: "5xchu",
				},
			},
			configMap: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						label.Cluster:      "5xchu",
						label.Organization: "giantswarm",
					},
					Name:      "5xchu-cluster-values",
					Namespace: "5xchu",
				},
			},
			expectedName: "giantswarm",
			expectedLabels: map[string]string{
				"giantswarm.io/cluster":      "5xchu",
				"giantswarm.io/managed-by":   "app-operator",
				"giantswarm.io/organization": "giantswarm",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0, 0)
			objs = append(objs, &tc.configMap)
			fakeClient := fake.NewSimpleClientset(objs...)
			c := Config{
				K8sClient: fakeClient,
				Logger:    microloggertest.New(),
			}
			newResource, err := New(c)
			if err != nil {
				t.Fatal("expected", nil, "got", err)
			}

			result, err := newResource.GetDesiredState(context.TODO(), tc.obj)
			if err != nil {
				t.Fatal("expected", nil, "got", err)
			}

			name := result.(*corev1.Namespace).Name
			if tc.expectedName != name {
				t.Fatalf("expected %q got %q", tc.expectedName, name)
			}

			labels := result.(*corev1.Namespace).Labels
			if !reflect.DeepEqual(tc.expectedLabels, labels) {
				t.Fatalf("expected %#v got %#v", tc.expectedLabels, labels)
			}
		})
	}
}
