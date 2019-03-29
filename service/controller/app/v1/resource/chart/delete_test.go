package chart

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
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
			name: "case 0: chart should be deleted",
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
		{
			name: "case 1: chart should not deleted",
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
					Labels: map[string]string{
						"app": "prometheus-1",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
			expectedChart: nil,
		},
	}

	c := Config{
		G8sClient: fake.NewSimpleClientset(),
		Logger:    microloggertest.New(),

		ChartNamespace: "giantswarm",
		ProjectName:    "app-operator",
		WatchNamespace: "default",
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

			chart, err := key.ToChart(result)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if result != nil {
				if !reflect.DeepEqual(chart, tc.expectedChart) {
					t.Fatalf("want matching chart \n %s", cmp.Diff(result, tc.expectedChart))
				}
			}
		})
	}
}
