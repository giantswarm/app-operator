package chart

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/service/controller/app/v1/kubeconfig"
)

func TestResource_newCreateChange(t *testing.T) {
	type fields struct {
		g8sClient      versioned.Interface
		k8sClient      kubernetes.Interface
		kubeConfig     *kubeconfig.KubeConfig
		logger         micrologger.Logger
		watchNamespace string
	}
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
					Name: "kubernetes-prometheus",
					Labels: map[string]string{
						"app": "prometheus",
					},
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
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubernetes-prometheus",
					Labels: map[string]string{
						"app": "prometheus",
					},
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
			name: "case 1: chart already existed",
			currentResource: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubernetes-prometheus",
					Labels: map[string]string{
						"app": "prometheus",
					},
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
			desiredResource: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubernetes-prometheus",
					Labels: map[string]string{
						"app": "prometheus",
					},
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
			expectedChart: &v1alpha1.Chart{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var err error
			var kc *kubeconfig.KubeConfig
			{
				c := kubeconfig.Config{
					G8sClient: fake.NewSimpleClientset(),
					K8sClient: k8sfake.NewSimpleClientset(),
					Logger:    microloggertest.New(),
				}

				kc, err = kubeconfig.New(c)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}
			}
			c := Config{
				G8sClient:      fake.NewSimpleClientset(),
				K8sClient:      k8sfake.NewSimpleClientset(),
				KubeConfig:     kc,
				Logger:         microloggertest.New(),
				WatchNamespace: "default",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			got, err := r.newCreateChange(context.TODO(), tt.currentResource, tt.desiredResource)
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
