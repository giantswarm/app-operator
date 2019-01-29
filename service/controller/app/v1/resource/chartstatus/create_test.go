package chartstatus

import (
	"context"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/service/controller/app/v1/kubeconfig"
)

func TestResource_EnsureCreated(t *testing.T) {
	tests := []struct {
		name         string
		obj          *v1alpha1.App
		chart        *v1alpha1.Chart
		errorMatcher func(error) bool
	}{
		{
			name: "case 0: update status flow",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "prometheus",
					Namespace: "monitoring",
					Release:   "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "giant-swarm-secret",
							Namespace: "giantswarm",
						},
					},
				},
			},
			chart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config:     v1alpha1.ChartSpecConfig{},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/prometheus-1.0.0.tgz",
				},
				Status: v1alpha1.ChartStatus{
					Status:       "DEPLOYED",
					LastDeployed: v1alpha1.DeepCopyTime{time.Date(2019, 1, 1, 13, 0, 0, 0, time.UTC)},
				},
			},
		},
		{
			name: "case 1: status not updated",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "prometheus",
					Namespace: "monitoring",
					Release:   "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "giant-swarm-secret",
							Namespace: "giantswarm",
						},
					},
				},
				Status: v1alpha1.AppStatus{
					Status:       "DEPLOYED",
					LastDeployed: v1alpha1.DeepCopyTime{time.Date(2019, 1, 1, 13, 0, 0, 0, time.UTC)},
				},
			},
			chart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config:     v1alpha1.ChartSpecConfig{},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/prometheus-1.0.0.tgz",
				},
				Status: v1alpha1.ChartStatus{
					Status:       "DEPLOYED",
					LastDeployed: v1alpha1.DeepCopyTime{time.Date(2019, 1, 1, 13, 0, 0, 0, time.UTC)},
				},
			},
		},
		{
			name: "case 2: cannot find chart",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "prometheus",
					Namespace: "monitoring",
					Release:   "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "giant-swarm-secret",
							Namespace: "giantswarm",
						},
					},
				},
				Status: v1alpha1.AppStatus{
					Status:       "DEPLOYED",
					LastDeployed: v1alpha1.DeepCopyTime{time.Date(2019, 1, 1, 13, 0, 0, 0, time.UTC)},
				},
			},
			errorMatcher: IsNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			var err error

			objs := make([]runtime.Object, 0, 0)
			if tc.obj != nil {
				objs = append(objs, tc.obj)
			}
			if tc.chart != nil {
				objs = append(objs, tc.chart)
			}

			g8sClient := fake.NewSimpleClientset(objs...)

			var kc *kubeconfig.KubeConfig
			{
				c := kubeconfig.Config{
					G8sClient: g8sClient,
					K8sClient: k8sfake.NewSimpleClientset(),
					Logger:    microloggertest.New(),
				}

				kc, err = kubeconfig.New(c)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}
			}

			c := Config{
				G8sClient:  g8sClient,
				K8sClient:  k8sfake.NewSimpleClientset(),
				KubeConfig: kc,
				Logger:     microloggertest.New(),

				WatchNamespace: "default",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			err = r.EnsureCreated(context.TODO(), tc.obj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if err == nil && tc.errorMatcher == nil {
				app, err := g8sClient.ApplicationV1alpha1().Apps(tc.obj.Namespace).Get(tc.obj.Name, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}
				if app.Status.LastDeployed != tc.chart.Status.LastDeployed {
					t.Fatalf("app.status.LastDeployed == %#v, want %#v", app.Status.LastDeployed, tc.chart.Status.LastDeployed)
				}
				if app.Status.Status != tc.chart.Status.Status {
					t.Fatalf("app.status.Status == %#v, want %#v", app.Status.Status, tc.chart.Status.Status)
				}
			}

		})
	}
}
