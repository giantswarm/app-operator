package chart

import (
	"context"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/resourcecanceledcontext"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/app-operator/v7/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v7/service/internal/indexcache/indexcachetest"
)

func Test_CordonUntil(t *testing.T) {
	tests := []struct {
		canceled bool
		error    error
		name     string
		obj      *v1alpha1.App
	}{
		{
			canceled: false,
			name: "flawless flow, not cordoned",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                                "test-app",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "flux",
					},
					Name:      "test-app",
					Namespace: "default",
				},
			},
		},
		{
			canceled: true,
			name: "flawless flow, cordon holds",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotation.AppOperatorCordonUntil: "2030-01-02T15:04:05Z",
					},
					Labels: map[string]string{
						"app":                                "test-app",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "flux",
					},
					Name:      "test-app",
					Namespace: "default",
				},
			},
		},
		{
			canceled: false,
			name: "flawless flow, cordon expired",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotation.AppOperatorCordonUntil: "2021-01-02T15:04:05Z",
					},
					Labels: map[string]string{
						"app":                                "test-app",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "flux",
					},
					Name:      "test-app",
					Namespace: "default",
				},
			},
		},
		{
			canceled: true,
			error: parseError("2030-01-02"),
			name: "cordon set with bad time format",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotation.AppOperatorCordonUntil: "2030-01-02",
					},
					Labels: map[string]string{
						"app":                                "test-app",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "flux",
					},
					Name:      "test-app",
					Namespace: "default",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := runtime.NewScheme()
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.AppList{})

			c := Config{
				IndexCache: indexcachetest.New(indexcachetest.Config{
					GetIndexResponse: nil,
				}),

				Logger:        microloggertest.New(),
				CtrlClient:    fake.NewClientBuilder().WithScheme(s).Build(), //nolint:staticcheck
				DynamicClient: dynamicfake.NewSimpleDynamicClient(s),

				ChartNamespace:               "giantswarm",
				DependencyWaitTimeoutMinutes: 30,
			}

			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var ctx context.Context
			{
				s := runtime.NewScheme()
				s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Chart{}, &v1alpha1.ChartList{})
				config := k8sclienttest.ClientsConfig{
					CtrlClient: fake.NewClientBuilder().WithScheme(s).Build(), //nolint:staticcheck
					K8sClient:  clientgofake.NewSimpleClientset(),
				}
				client := k8sclienttest.NewClients(config)

				c := controllercontext.Context{
					Clients: controllercontext.Clients{
						K8s: client,
					},
				}
				ctx = controllercontext.NewContext(context.Background(), c)
				ctx = resourcecanceledcontext.NewContext(ctx, make(chan struct{}))
			}

			_, err = r.GetCurrentState(ctx, tc.obj)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if tc.canceled != resourcecanceledcontext.IsCanceled(ctx) {
				t.Fatalf("canceled is %#v, want %#v", resourcecanceledcontext.IsCanceled(ctx), tc.canceled)
			}
		})
	}
}

func parseError(t string) error {
	_, err := time.Parse(time.RFC3339, t)

	return err
}
