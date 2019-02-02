package chart

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Resource_newCreateChange(t *testing.T) {
	tests := []struct {
		name            string
		currentResource *v1alpha1.Chart
		desiredResource *v1alpha1.Chart
		expectedChart   *v1alpha1.Chart
	}{
		{
			name:            "case 0: new chart should be created",
			currentResource: &v1alpha1.Chart{},
			desiredResource: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "prometheus",
					Labels: map[string]string{
						"app": "prometheus",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "prometheus",
					Labels: map[string]string{
						"app": "prometheus",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
		},
		{
			name: "case 1: chart already exist",
			currentResource: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "prometheus",
					Labels: map[string]string{
						"app": "prometheus",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
			desiredResource: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "prometheus",
					Labels: map[string]string{
						"app": "prometheus",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
			expectedChart: &v1alpha1.Chart{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{
				G8sClient:  fake.NewSimpleClientset(),
				Logger: microloggertest.New(),

				ProjectName:    "app-operator",
				WatchNamespace: "default",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			got, err := r.newCreateChange(context.Background(), tt.currentResource, tt.desiredResource)
			if err != nil {
				t.Errorf("Resource.newCreateChange() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.expectedChart) {
				t.Errorf("Resource.newCreateChange() = %v, want %v", got, tt.expectedChart)
			}
		})
	}
}
