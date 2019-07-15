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

func Test_Resource_newUpdateChange(t *testing.T) {
	tests := []struct {
		name            string
		currentResource *v1alpha1.Chart
		desiredResource *v1alpha1.Chart
		expectedChart   *v1alpha1.Chart
	}{
		{
			name: "case 0: chart should be updated",
			currentResource: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name: "prometheus",
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.1.tgz",
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
			name: "case 1: chart should not be update",
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
			expectedChart: &v1alpha1.Chart{},
		},
		{
			name: "case 1: chart should be update for cordon annotations",
			currentResource: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/values-md5-checksum": "52668444a530c7b296f9c620f8cb2632",
					},
					Name: "prometheus",
				},
				Spec: v1alpha1.ChartSpec{
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
				},
			},
			desiredResource: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/cordon-reason": "chart maintenance",
						"chart-operator.giantswarm.io/cordon-until":  "2019-12-31T23:59:59Z",
					},
					Name: "prometheus",
				},
				Spec: v1alpha1.ChartSpec{
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
				},
			},
			expectedChart: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/cordon-reason": "chart maintenance",
						"chart-operator.giantswarm.io/cordon-until":  "2019-12-31T23:59:59Z",
					},
					Name: "prometheus",
				},
				Spec: v1alpha1.ChartSpec{
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
				},
			},
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
			result, err := r.newUpdateChange(context.Background(), tc.currentResource, tc.desiredResource)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
				return
			}

			if !reflect.DeepEqual(result, tc.expectedChart) {
				t.Fatalf("want matching chart \n %s", cmp.Diff(result, tc.expectedChart))
			}
		})
	}
}
