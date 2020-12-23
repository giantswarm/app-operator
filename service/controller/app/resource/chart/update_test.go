package chart

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned/fake"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
)

func Test_Resource_GetUpdateState(t *testing.T) {
	tests := []struct {
		name          string
		currentChart  *v1alpha1.Chart
		desiredChart  *v1alpha1.Chart
		expectedChart *v1alpha1.Chart
		error         bool
	}{
		{
			name: "case 0: flawless flow",
			currentChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-cool-prometheus",
					Namespace:       "giantswarm",
					ResourceVersion: "12345",
					UID:             "51eeec1d-3716-4006-92b4-e7e99f8ab311",
					Annotations: map[string]string{
						"giantswarm.io/sample": "it should be deleted",
					},
					Labels: map[string]string{
						"giantswarm.io/managed-by": "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "my-cool-prometheus-chart-values",
							Namespace: "giantswarm",
						},
					},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/prometheus-1.0.0.tgz",
					Version:    "0.0.9",
				},
			},
			desiredChart: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/webhook-url": "http://webhook/status/default/my-cool-prometheus",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "my-cool-prometheus-chart-values",
							Namespace: "giantswarm",
						},
					},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
			expectedChart: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-cool-prometheus",
					Namespace:       "giantswarm",
					ResourceVersion: "12345",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/webhook-url": "http://webhook/status/default/my-cool-prometheus",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "my-cool-prometheus-chart-values",
							Namespace: "giantswarm",
						},
					},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
		},
		{
			name: "case 1: same chart",
			currentChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/webhook-url": "http://webhook/status/default/my-cool-prometheus",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "my-cool-prometheus-chart-values",
							Namespace: "giantswarm",
						},
					},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
			desiredChart: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/webhook-url": "http://webhook/status/default/my-cool-prometheus",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "my-cool-prometheus-chart-values",
							Namespace: "giantswarm",
						},
					},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
			expectedChart: &v1alpha1.Chart{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0)

			c := Config{
				Logger: microloggertest.New(),

				ChartNamespace: "giantswarm",
				WebhookBaseURL: "http://webhook",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var ctx context.Context
			{
				config := k8sclienttest.ClientsConfig{
					G8sClient: fake.NewSimpleClientset(),
					K8sClient: clientgofake.NewSimpleClientset(objs...),
				}
				client := k8sclienttest.NewClients(config)

				c := controllercontext.Context{
					Clients: controllercontext.Clients{
						K8s: client,
					},
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			result, err := r.newUpdateChange(ctx, tc.currentChart, tc.desiredChart)
			switch {
			case err != nil && !tc.error:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.error:
				t.Fatalf("error == nil, want non-nil")
			}

			if err == nil && !tc.error {
				chart, err := toChart(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(chart.ObjectMeta, tc.expectedChart.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(chart.ObjectMeta, tc.expectedChart.ObjectMeta))
				}

				if !reflect.DeepEqual(chart.Spec, tc.expectedChart.Spec) {
					t.Fatalf("want matching spec \n %s", cmp.Diff(chart.Spec, tc.expectedChart.Spec))
				}

				if !reflect.DeepEqual(chart.TypeMeta, tc.expectedChart.TypeMeta) {
					t.Fatalf("want matching typemeta \n %s", cmp.Diff(chart.TypeMeta, tc.expectedChart.TypeMeta))
				}
			}
		})
	}
}
