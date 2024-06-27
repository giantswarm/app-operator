package helmreleasestatus

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stest "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/app-operator/v6/pkg/project"
)

func Test_doWatchStatus_CAPI(t *testing.T) {
	tests := []struct {
		app            *v1alpha1.App
		expectedStatus v1alpha1.AppStatus
		helmRelease    *helmv2.HelmRelease
		name           string
	}{
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                   "hello-world",
						"giantswarm.io/cluster": "1234",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "hello-world-app",
					Namespace: "default",
					Version:   "1.0.0",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "12345-kubeconfig",
							Namespace: "org-test",
						},
					},
				},
			},
			expectedStatus: v1alpha1.AppStatus{
				AppVersion: "1.0.0",
				Release: v1alpha1.AppStatusRelease{
					Reason: "Helm install succeeded",
					Status: helmclient.StatusDeployed,
				},
				Version: "1.0.0",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						label.Cluster:   "1234",
						label.ManagedBy: project.Name(),
					},
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    fluxmeta.ReadyCondition,
							Message: "release reconciliation succeeded",
							Reason:  helmv2.ReconciliationSucceededReason,
						},
						metav1.Condition{
							Type:    helmv2.ReleasedCondition,
							Message: "Helm install succeeded",
							Reason:  helmv2.InstallSucceededReason,
						},
					},
				},
			},
			name: "flawless update status",
		},
	}

	for c, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", c, tc.name), func(t *testing.T) {
			objs := []runtime.Object{
				tc.app,
				tc.helmRelease,
			}

			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)
			_ = helmv2.AddToScheme(scheme)
			_ = sourcev1.AddToScheme(scheme)

			ctrlClient := clientfake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objs...).
				WithStatusSubresource([]client.Object{tc.app, tc.helmRelease}...).
				Build()

			dynClient := dynamicfake.NewSimpleDynamicClient(scheme, objs...)

			clientConfig := k8sclienttest.ClientsConfig{
				CtrlClient: ctrlClient,
				DynClient:  dynClient,
			}

			config := HelmReleaseStatusWatcherConfig{
				Logger:    microloggertest.New(),
				K8sClient: k8sclienttest.NewClients(clientConfig),

				PodNamespace:      "org-test",
				UniqueApp:         false,
				WorkloadClusterID: "1234",
			}

			r, err := NewHelmReleaseStatusWatcher(config)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			ctx := context.Background()
			go r.doWatchStatus(ctx, dynClient)

			watcher := watch.NewFake()
			dynClient.PrependWatchReactor("helmreleases", k8stest.DefaultWatchReactor(watcher, nil))

			defer watcher.Stop()
			watcher.Add(tc.helmRelease)

			o := func() error {
				var app v1alpha1.App

				err = ctrlClient.Get(ctx,
					types.NamespacedName{Name: tc.app.Name, Namespace: tc.app.Namespace},
					&app)
				if err != nil {
					return microerror.Mask(err)
				}

				if !reflect.DeepEqual(app.Status, tc.expectedStatus) {
					return fmt.Errorf("want matching statuses \n %s", cmp.Diff(app.Status, tc.expectedStatus))
				}

				return nil
			}

			b := backoff.NewExponentialBackOff()
			b.MaxElapsedTime = 2 * time.Second
			b.InitialInterval = 100 * time.Millisecond
			b.MaxInterval = b.InitialInterval
			b.RandomizationFactor = 0.0

			err = backoff.Retry(o, b)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}
		})
	}
}
