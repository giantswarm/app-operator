package chart

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func Test_Resource_GetCurrentState(t *testing.T) {
	tests := []struct {
		name          string
		obj           *v1alpha1.App
		returnedChart *v1alpha1.Chart
		errorMatcher  func(error) bool
	}{
		{
			name: "case 0: chart already created",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "giantswarm",
					Version: "1.0.0",
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
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "",
							Namespace: "",
						},
					},
					Name:      "kubernetes-prometheus",
					Namespace: "monitoring",
				},
			},
			returnedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubernetes-prometheus",
					Namespace: "default",
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:            "giant-swarm-config",
							Namespace:       "giantswarm",
							ResourceVersion: "",
						},
						Secret: v1alpha1.ChartSpecConfigSecret{
							Name:            "giant-swarm-secret",
							Namespace:       "giantswarm",
							ResourceVersion: "",
						},
					},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.com/app-catalog/kubernetes-prometheus-1.0.0.tgz",
				},
			},
		},
		{
			name: "case 1: chart not found",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "giantswarm",
					Version: "1.0.0",
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
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "",
							Namespace: "",
						},
					},
					Name:      "kubernetes-prometheus",
					Namespace: "monitoring",
				},
			},
			returnedChart: nil,
		}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0, 0)
			if tc.returnedChart != nil {
				objs = append(objs, tc.returnedChart)
			}

			g8sClient := fake.NewSimpleClientset(objs...)

			var err error

			var kc kubeconfig.Interface
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

				ProjectName:    "app-operator",
				WatchNamespace: "default",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			ctlConfig := controllercontext.Context{
				G8sClient: g8sClient,
			}
			ctx := controllercontext.NewContext(context.TODO(), ctlConfig)

			result, err := r.GetCurrentState(ctx, tc.obj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if err == nil && tc.errorMatcher == nil {
				if result != nil {
					chart, err := key.ToChart(result)
					if err != nil {
						t.Fatalf("error == %#v, want nil", err)
					}

					if !reflect.DeepEqual(chart, *tc.returnedChart) {
						t.Fatalf("Chart == %#v, want %#v", chart, tc.returnedChart)
					}
				}
			}
		})
	}
}
