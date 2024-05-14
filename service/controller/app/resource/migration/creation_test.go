package migration

import (
	"context"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func Test_EnsureCreated(t *testing.T) {
	tests := []struct {
		app             *v1alpha1.App
		chart           *v1alpha1.Chart
		deploymentGone  bool
		nativeResources []runtime.Object
		name            string
	}{
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "1234-test-app",
					Namespace: "org-test",
				},
			},
			chart: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{
						"operatorkit.giantswarm.io/chart-operator-chart",
					},
					Name:      "test-app",
					Namespace: "giantswarm",
				},
			},
			deploymentGone: true,
			name:           "case 0: flawless flow, Chart Operator Deployment gone",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "1234-test-app",
					Namespace: "org-test",
				},
			},
			chart: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{
						"operatorkit.giantswarm.io/chart-operator-chart",
					},
					Name:      "test-app",
					Namespace: "giantswarm",
				},
			},
			nativeResources: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "chart-operator",
						Namespace: "giantswarm",
					},
				},
			},
			name: "case 1: flawless flow, Chart Operator Deployment left around",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "1234-test-app",
					Namespace: "org-test",
				},
			},
			chart: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{
						"operatorkit.giantswarm.io/chart-operator-chart",
					},
					Name:      "test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "test-app-chart-values",
							Namespace: "giantswarm",
						},
						Secret: v1alpha1.ChartSpecConfigSecret{
							Name:      "test-app-chart-secrets",
							Namespace: "giantswarm",
						},
					},
				},
			},
			nativeResources: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-chart-values",
						Namespace: "giantswarm",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-chart-secrets",
						Namespace: "giantswarm",
					},
				},
			},
			name: "case 2: flawless flow with Chart CR configuration",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			customObjs := []runtime.Object{
				tc.chart,
			}

			c := Config{
				Logger:     microloggertest.New(),
				CtrlClient: fake.NewFakeClient(), //nolint:staticcheck

				ChartNamespace:    "giantswarm",
				WorkloadClusterID: "1234",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var ctx context.Context
			var client *k8sclienttest.Clients
			{
				s := runtime.NewScheme()
				s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Chart{})

				config := k8sclienttest.ClientsConfig{
					CtrlClient: fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(customObjs...).Build(),
					K8sClient:  clientgofake.NewSimpleClientset(tc.nativeResources...),
				}
				client = k8sclienttest.NewClients(config)

				c := controllercontext.Context{
					MigrationClients: controllercontext.Clients{
						K8s: client,
					},
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			err = r.EnsureCreated(ctx, tc.app)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var chart v1alpha1.Chart
			err = client.CtrlClient().Get(
				ctx,
				types.NamespacedName{Name: tc.chart.Name, Namespace: r.chartNamespace},
				&chart,
			)
			if !tc.deploymentGone && err == nil {
				if chart.Annotations[ChartOperatorPaused] != "true" {
					t.Fatalf("want pause annotation set, got %#q", chart.Annotations[ChartOperatorPaused])
				}
			}
			if tc.deploymentGone && err == nil {
				t.Fatalf("want notFoundError, got nil")
			}
			if err != nil && !apierrors.IsNotFound(err) {
				t.Fatalf("error == %#v, want notFoundError", err)
			}

			for _, o := range tc.nativeResources {
				switch obj := o.(type) {
				case *appsv1.Deployment:
					continue
				case *corev1.ConfigMap:
					_, err = client.K8sClient().CoreV1().ConfigMaps(obj.Namespace).Get(ctx, obj.Name, metav1.GetOptions{})
				case *corev1.Secret:
					_, err = client.K8sClient().CoreV1().Secrets(obj.Namespace).Get(ctx, obj.Name, metav1.GetOptions{})
				default:
					continue
				}

				if err == nil {
					t.Fatalf("got nil, want notFoundError")
				}
				if err != nil && !apierrors.IsNotFound(err) {
					t.Fatalf("error == %#v, want notFoundError", err)
				}
			}
		})
	}
}
