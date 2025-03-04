package tcnamespace

import (
	"context"
	"testing"
	"fmt"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v7/service/controller/app/controllercontext"
)

func Test_Create(t *testing.T) {
	tests := []struct {
		existsBefore bool
		name         string
		obj          *v1alpha1.App
	}{
		{
			existsBefore: false,
			name:         "create namespace",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "wc-chart-operator",
					Namespace: "org-org",
					Labels: map[string]string{
						"giantswarm.io/cluster": "wc",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "control-plane-catalog",
					Name:      "chart-operator",
					Namespace: "giantswarm",
					Version:   "1.0.0",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "wc-kubeconfig",
							Namespace: "org-org",
						},
					},
				},
			},
		},
		{
			existsBefore: true,
			name:         "namespace already exists",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "wc-chart-operator",
					Namespace: "org-org",
					Labels: map[string]string{
						"giantswarm.io/cluster": "wc",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "control-plane-catalog",
					Name:      "chart-operator",
					Namespace: "giantswarm",
					Version:   "1.0.0",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "wc-kubeconfig",
							Namespace: "org-org",
						},
					},
				},
			},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			c := Config{
				Logger: microloggertest.New(),
			}

			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			config := k8sclienttest.ClientsConfig{
				K8sClient: clientgofake.NewSimpleClientset(),
			}
			client := k8sclienttest.NewClients(config)

			var ctx context.Context
			{
				c := controllercontext.Context{
					Clients: controllercontext.Clients{
						K8s: client,
					},
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			if tc.existsBefore {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: namespace,
					},
				}

				_, err = client.K8sClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}
			}

			err = r.EnsureCreated(ctx, tc.obj)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			_, err = client.K8sClient().CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}
		})
	}
}
