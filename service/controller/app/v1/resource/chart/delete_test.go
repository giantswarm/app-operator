package chart

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Resource_newDeleteChange(t *testing.T) {
	tests := []struct {
		name            string
		currentResource *v1alpha1.Chart
		desiredResource *v1alpha1.Chart
		expectedChart   *v1alpha1.Chart
	}{
		{
			name:            "case 0: empty current and desired, expected empty",
			currentResource: &v1alpha1.Chart{},
			desiredResource: &v1alpha1.Chart{},
			expectedChart:   &v1alpha1.Chart{},
		},
		{
			name: "case 1: chart should be deleted",
			currentResource: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name: "prometheus",
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
			desiredResource: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name: "prometheus",
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
			expectedChart: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name: "prometheus",
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
		},
	}

	c := Config{
		G8sClient: fake.NewSimpleClientset(),
		Logger:    microloggertest.New(),

		ChartNamespace: "giantswarm",
	}
	r, err := New(c)
	if err != nil {
		t.Fatalf("error == %#v, want nil", err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := r.newDeleteChange(context.Background(), nil, tc.currentResource, tc.desiredResource)
			if err != nil {
				t.Fatalf("error = %v", err)
				return
			}

			chart, err := toChart(result)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if !reflect.DeepEqual(chart, tc.expectedChart) {
				t.Fatalf("want matching chart \n %s", cmp.Diff(result, tc.expectedChart))
			}
		})
	}
}
