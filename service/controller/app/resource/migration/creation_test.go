package migration

import (
	"context"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
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
		app   *v1alpha1.App
		chart *v1alpha1.Chart
		name  string
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
			name: "case 0: flawless flow",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := []runtime.Object{
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
					CtrlClient: fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build(),
					K8sClient:  clientgofake.NewSimpleClientset(),
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
			if err != nil && !apierrors.IsNotFound(err) {
				t.Fatalf("error == %#v, want notFoundError", err)
			}
		})
	}
}
