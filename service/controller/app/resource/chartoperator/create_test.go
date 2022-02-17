package chartoperator

import (
	"context"
	"fmt"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/values"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/spf13/afero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
)

func Test_Resource_triggerReconciliation(t *testing.T) {
	tests := []struct {
		name             string
		apps             []*v1alpha1.App
		chartoperator    *v1alpha1.App
		charts           []*v1alpha1.Chart
		expectAnnotation map[string]bool
	}{
		{
			name:          "flawless cluster namespace",
			chartoperator: newApp("chart-operator", "chart-operator", "1abc2", "", false),
			apps: []*v1alpha1.App{
				newApp("app-operator-1abc2", "app-operator", "1abc2", "", true),
				newApp("cert-exporter", "cert-exporter", "1abc2", "", false),
				newApp("cert-operator", "cert-operator", "1abc2", "", false),
				newApp("cluster-autoscaler", "cluster-autoscaler", "1abc2", "", false),
				newApp("kiam", "kiam", "1abc2", "", false),
			},
			charts: []*v1alpha1.Chart{
				newChart("kiam", "giantswarm"),
			},
			expectAnnotation: map[string]bool{
				"app-operator-1abc2": false,
				"cert-exporter":      true,
				"cert-operator":      true,
				"cluster-autoscaler": true,
				"kiam":               false,
			},
		},
		{
			name:          "flawless organization namespace",
			chartoperator: newApp("chart-operator", "chart-operator", "1abc2", "org-acme", false),
			apps: []*v1alpha1.App{
				newApp("1abc2-app-operator", "app-operator", "1abc2", "org-acme", true),
				newApp("1abc2-cert-exporter", "cert-exporter", "1abc2", "org-acme", false),
				newApp("1abc2-cert-operator", "cert-operator", "1abc2", "org-acme", false),
				newApp("1abc2-cluster-autoscaler", "cluster-autoscaler", "1abc2", "org-acme", false),
				newApp("1abc2-kiam", "kiam", "1abc2", "org-acme", false),
				// different cluster
				newApp("3def4-app-operator", "app-operator", "3def4", "org-acme", true),
				newApp("3def4-cert-exporter", "cert-exporter", "3def4", "org-acme", false),
				newApp("3def4-cert-operator", "cert-operator", "3def4", "org-acme", false),
				newApp("3def4-cluster-autoscaler", "cluster-autoscaler", "3def4", "org-acme", false),
				newApp("3def4-kiam", "kiam", "3def4", "org-acme", false),
			},
			charts: []*v1alpha1.Chart{
				newChart("kiam", "giantswarm"),
			},
			expectAnnotation: map[string]bool{
				"1abc2-app-operator":       false,
				"1abc2-cert-exporter":      true,
				"1abc2-cert-operator":      true,
				"1abc2-cluster-autoscaler": true,
				"1abc2-kiam":               false,
				// different cluster
				"3def4-app-operator":       false,
				"3def4-cert-exporter":      false,
				"3def4-cert-operator":      false,
				"3def4-cluster-autoscaler": false,
				"3def4-kiam":               false,
			},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("%d: %s", i, tc.name), func(t *testing.T) {
			var err error

			schemeBuilder := runtime.SchemeBuilder{
				v1alpha1.AddToScheme,
			}

			err = schemeBuilder.AddToScheme(scheme.Scheme)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var r *Resource
			{
				var objs []runtime.Object
				objs = append(objs, tc.chartoperator)
				for _, v := range tc.apps {
					objs = append(objs, v)
				}

				fakeCtrlClient := fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithRuntimeObjects(objs...).
					Build()

				fakeK8sClient := clientgofake.NewSimpleClientset()
				fakeLogger := microloggertest.New()

				var valuesService *values.Values
				{
					c := values.Config{
						K8sClient: fakeK8sClient,
						Logger:    fakeLogger,
					}

					valuesService, err = values.New(c)
					if err != nil {
						t.Fatalf("error == %#v, want nil", err)
					}
				}

				c := Config{
					CtrlClient: fakeCtrlClient,
					FileSystem: afero.NewMemMapFs(),
					K8sClient:  fakeK8sClient,
					Logger:     fakeLogger,

					ChartNamespace:    "giantswarm",
					WorkloadClusterID: "1abc2",
					Values:            valuesService,
				}

				r, err = New(c)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}
			}

			var ctx context.Context
			{
				var objs []runtime.Object
				for _, v := range tc.charts {
					objs = append(objs, v)
				}

				fakeCtrlClient := fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithRuntimeObjects(objs...).
					Build()

				fakeClient := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
					CtrlClient: fakeCtrlClient,
					K8sClient:  clientgofake.NewSimpleClientset(),
				})

				c := controllercontext.Context{
					Clients: controllercontext.Clients{
						K8s: fakeClient,
					},
				}

				ctx = controllercontext.NewContext(context.Background(), c)
			}

			err = r.triggerReconciliation(ctx, *tc.chartoperator)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var appList v1alpha1.AppList
			err = r.ctrlClient.List(ctx, &appList)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			for _, a := range appList.Items {
				_, ok := a.GetAnnotations()[annotation.AppOperatorTriggerReconciliation]
				expectedSet := tc.expectAnnotation[a.ObjectMeta.Name]
				if expectedSet != ok {
					t.Fatalf("%s: expected %t, got %t", a.ObjectMeta.Name, expectedSet, ok)
				}

				expectedNum := 1
				if expectedSet {
					expectedNum = 2
				}

				if expectedNum != len(a.GetAnnotations()) {
					t.Fatalf("%s: expected %d, got %d", a.ObjectMeta.Name, expectedNum, len(a.GetAnnotations()))
				}
			}
		})
	}
}

func newApp(crName, appName, cluster, organization string, inCluster bool) *v1alpha1.App {
	metaLabels := map[string]string{}
	namespace := cluster

	if organization != "" {
		metaLabels[label.Cluster] = cluster
		namespace = organization
	}

	c := &v1alpha1.App{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "application.giantswarm.io/v1alpha1",
			Kind:       "App",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"dummy-annotation": "dummy-value",
			},
			Labels:    metaLabels,
			Name:      crName,
			Namespace: namespace,
		},
		Spec: v1alpha1.AppSpec{
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: inCluster,
			},
			Name: appName,
		},
	}

	return c
}

func newChart(name, namespace string) *v1alpha1.Chart {
	c := &v1alpha1.Chart{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "application.giantswarm.io/v1alpha1",
			Kind:       "Chart",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	return c
}
