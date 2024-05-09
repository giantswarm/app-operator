package status

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/app-operator/v6/pkg/status"
	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func Test_EnsureCreated_Chart(t *testing.T) {
	tests := []struct {
		app            *v1alpha1.App
		chart          *v1alpha1.Chart
		contextStatus  *controllercontext.ChartStatus
		expectedStatus v1alpha1.AppStatus
		name           string
	}{
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
			chart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "hello-world-app",
					Namespace:  "default",
					TarballURL: "https://giantswarm.github.io/app-catalog/hello-world-app-1.0.0.tgz",
					Version:    "1.0.0",
				},
				Status: v1alpha1.ChartStatus{
					AppVersion: "0.23.0",
					Reason:     "",
					Release: v1alpha1.ChartStatusRelease{
						Status: helmclient.StatusDeployed,
					},
					Version: "1.0.0",
				},
			},
			expectedStatus: v1alpha1.AppStatus{
				AppVersion: "0.23.0",
				Release: v1alpha1.AppStatusRelease{
					Status: helmclient.StatusDeployed,
				},
				Version: "1.0.0",
			},
			name: "flawless from-Chart status creation",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
				Status: v1alpha1.AppStatus{
					AppVersion: "0.22.0",
					Release: v1alpha1.AppStatusRelease{
						Status: helmclient.StatusDeployed,
					},
					Version: "0.0.9",
				},
			},
			chart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "hello-world-app",
					Namespace:  "default",
					TarballURL: "https://giantswarm.github.io/app-catalog/hello-world-app-1.0.0.tgz",
					Version:    "1.0.0",
				},
				Status: v1alpha1.ChartStatus{
					AppVersion: "0.23.0",
					Reason:     "",
					Release: v1alpha1.ChartStatusRelease{
						Status: helmclient.StatusDeployed,
					},
					Version: "1.0.0",
				},
			},
			expectedStatus: v1alpha1.AppStatus{
				AppVersion: "0.23.0",
				Release: v1alpha1.AppStatusRelease{
					Status: helmclient.StatusDeployed,
				},
				Version: "1.0.0",
			},
			name: "flawless from-Chart status update",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
			contextStatus: &controllercontext.ChartStatus{
				Reason: "problem found",
				Status: status.AppNotFoundStatus,
			},
			expectedStatus: v1alpha1.AppStatus{
				AppVersion: "",
				Release: v1alpha1.AppStatusRelease{
					Reason: "problem found",
					Status: status.AppNotFoundStatus,
				},
				Version: "",
			},
			name: "flawless from-Context status creation",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
				Status: v1alpha1.AppStatus{
					AppVersion: "0.22.0",
					Release: v1alpha1.AppStatusRelease{
						Status: helmclient.StatusDeployed,
					},
					Version: "0.0.9",
				},
			},
			contextStatus: &controllercontext.ChartStatus{
				Reason: "problem found",
				Status: status.AppNotFoundStatus,
			},
			expectedStatus: v1alpha1.AppStatus{
				AppVersion: "",
				Release: v1alpha1.AppStatusRelease{
					Reason: "problem found",
					Status: status.AppNotFoundStatus,
				},
				Version: "",
			},
			name: "flawless from-Context status update",
		},
	}
	for c, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", c, tc.name), func(t *testing.T) {
			mcObjs := make([]client.Object, 0)
			if tc.app != nil {
				mcObjs = append(mcObjs, tc.app)
			}

			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)

			resourceClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(mcObjs...).
				WithStatusSubresource(mcObjs...).
				Build()

			c := Config{
				Logger:     microloggertest.New(),
				CtrlClient: resourceClient,

				ChartNamespace:    "giantswarm",
				WorkloadClusterID: "12345",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var ctx context.Context
			{
				wcObjs := make([]runtime.Object, 0)
				if tc.chart != nil {
					wcObjs = append(wcObjs, tc.chart)
				}

				config := k8sclienttest.ClientsConfig{
					K8sClient: clientgofake.NewSimpleClientset(),
				}

				config.CtrlClient = fake.NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(wcObjs...).
					Build()

				client := k8sclienttest.NewClients(config)

				c := controllercontext.Context{
					Clients: controllercontext.Clients{
						K8s: client,
					},
					Catalog: v1alpha1.Catalog{},
				}

				if tc.contextStatus != nil {
					c.Status = controllercontext.Status{
						ChartStatus: *tc.contextStatus,
					}
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			err = r.EnsureCreated(ctx, tc.app)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var updated v1alpha1.App
			err = r.ctrlClient.Get(
				ctx,
				types.NamespacedName{Name: tc.app.Name, Namespace: tc.app.Namespace},
				&updated,
			)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if !reflect.DeepEqual(key.AppStatus(updated), tc.expectedStatus) {
				t.Fatalf("want matching statuses \n %s", cmp.Diff(key.AppStatus(updated), tc.expectedStatus))
			}
		})
	}
}

func Test_EnsureCreated_HelmRelease(t *testing.T) {
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
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
			name: "Released condition present, installation successfull",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
					Reason: "Reconciliation in progress",
					Status: status.PendingStatus,
				},
				Version: "1.0.0",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    fluxmeta.ReadyCondition,
							Message: "Reconciliation in progress",
							Reason:  fluxmeta.ProgressingReason,
						},
					},
				},
			},
			name: "Released condition missing, reconciliation in process",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
					Reason: "HelmChart 'org-test/org-test-hello-world' is not ready",
					Status: status.ChartPullFailedStatus,
				},
				Version: "1.0.0",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    fluxmeta.ReadyCondition,
							Message: "HelmChart 'org-test/org-test-hello-world' is not ready",
							Reason:  helmv2.ArtifactFailedReason,
						},
					},
				},
			},
			name: "Released condition missing, artifact failure",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
					Reason: "Initialization failure",
					Status: helmclient.StatusFailed,
				},
				Version: "1.0.0",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    fluxmeta.ReadyCondition,
							Message: "Initialization failure",
							Reason:  helmv2.InitFailedReason,
						},
					},
				},
			},
			name: "Released condition missing, initialization failure",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
					Reason: "Helm install failed: unable to build kubernetes objects from release manifest: error validating \"\": error validating data: apiVersion not set",
					Status: status.InvalidManifestStatus,
				},
				Version: "1.0.0",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    fluxmeta.ReadyCondition,
							Message: "Helm install failed",
							Reason:  helmv2.ReconciliationFailedReason,
						},
						metav1.Condition{
							Type:    helmv2.ReleasedCondition,
							Message: "Helm install failed: unable to build kubernetes objects from release manifest: error validating \"\": error validating data: apiVersion not set",
							Reason:  helmv2.InstallFailedReason,
						},
					},
				},
			},
			name: "Released condition present, invalid manifest failure",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
					Reason: "Helm install failed: values don't meet the specifications of the schema(s) in the following chart(s)",
					Status: status.ValuesSchemaViolation,
				},
				Version: "1.0.0",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    fluxmeta.ReadyCondition,
							Message: "Helm install failed",
							Reason:  helmv2.ReconciliationFailedReason,
						},
						metav1.Condition{
							Type:    helmv2.ReleasedCondition,
							Message: "Helm install failed: values don't meet the specifications of the schema(s) in the following chart(s)",
							Reason:  helmv2.InstallFailedReason,
						},
					},
				},
			},
			name: "Released condition present, values schema violation",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
					Reason: "Helm install failed: rendered manifests contain a resource that already exists. Unable to continue with install",
					Status: status.AlreadyExistsStatus,
				},
				Version: "1.0.0",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    fluxmeta.ReadyCondition,
							Message: "Helm install failed",
							Reason:  helmv2.ReconciliationFailedReason,
						},
						metav1.Condition{
							Type:    helmv2.ReleasedCondition,
							Message: "Helm install failed: rendered manifests contain a resource that already exists. Unable to continue with install",
							Reason:  helmv2.InstallFailedReason,
						},
					},
				},
			},
			name: "Released condition present, resource already exists",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
					Reason: "Helm install failed: error validating data",
					Status: status.ValidationFailedStatus,
				},
				Version: "1.0.0",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    fluxmeta.ReadyCondition,
							Message: "Helm install failed",
							Reason:  helmv2.ReconciliationFailedReason,
						},
						metav1.Condition{
							Type:    helmv2.ReleasedCondition,
							Message: "Helm install failed: error validating data",
							Reason:  helmv2.InstallFailedReason,
						},
					},
				},
			},
			name: "Released condition present, validation failure",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
					Reason: "Helm install failed: release name \"wrong_name\": invalid release name, must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$ and the length must not be longer than 53",
					Status: status.ReleaseNotInstalledStatus,
				},
				Version: "1.0.0",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    fluxmeta.ReadyCondition,
							Message: "Helm install failed",
							Reason:  helmv2.ReconciliationFailedReason,
						},
						metav1.Condition{
							Type:    helmv2.ReleasedCondition,
							Message: "Helm install failed: release name \"wrong_name\": invalid release name, must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$ and the length must not be longer than 53",
							Reason:  helmv2.InstallFailedReason,
						},
					},
				},
			},
			name: "Released condition present, validation failure",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
					Reason: "Helm install failed: this is unknown status",
					Status: helmclient.StatusFailed,
				},
				Version: "1.0.0",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    fluxmeta.ReadyCondition,
							Message: "Helm install failed",
							Reason:  helmv2.ReconciliationFailedReason,
						},
						metav1.Condition{
							Type:    helmv2.ReleasedCondition,
							Message: "Helm install failed: this is unknown status",
							Reason:  helmv2.InstallFailedReason,
						},
					},
				},
			},
			name: "Released condition present, failure with unrecognizable status",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
				AppVersion: "",
				Release: v1alpha1.AppStatusRelease{
					Reason: "",
					Status: "",
				},
				Version: "",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    helmv2.TestSuccessCondition,
							Message: "Helm test failed",
							Reason:  helmv2.TestFailedReason,
						},
					},
				},
			},
			name: "Released condition present, failure with unrecognizable status",
		},
		{
			app: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
					Labels: map[string]string{
						"app":                                "hello-world",
						"app-operator.giantswarm.io/version": "7.0.0",
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
				AppVersion: "",
				Release: v1alpha1.AppStatusRelease{
					Reason: "",
					Status: "",
				},
				Version: "",
			},
			helmRelease: &helmv2.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "org-test",
				},
				Status: helmv2.HelmReleaseStatus{
					LastAppliedRevision: "1.0.0",
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    helmv2.TestSuccessCondition,
							Message: "Helm test failed",
							Reason:  helmv2.TestFailedReason,
						},
					},
				},
			},
			name: "Released condition present, no conditions",
		},
	}
	for c, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", c, tc.name), func(t *testing.T) {
			mcObjs := make([]client.Object, 0)
			if tc.app != nil {
				mcObjs = append(mcObjs, tc.app)
			}

			if tc.helmRelease != nil {
				mcObjs = append(mcObjs, tc.helmRelease)
			}

			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)
			_ = helmv2.AddToScheme(scheme)

			resourceClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(mcObjs...).
				WithStatusSubresource(mcObjs...).
				Build()

			c := Config{
				Logger:     microloggertest.New(),
				CtrlClient: resourceClient,

				ChartNamespace:        "giantswarm",
				HelmControllerBackend: true,
				WorkloadClusterID:     "12345",
			}

			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var ctx context.Context
			{
				config := k8sclienttest.ClientsConfig{
					K8sClient: clientgofake.NewSimpleClientset(),
				}
				config.CtrlClient = resourceClient

				client := k8sclienttest.NewClients(config)

				c := controllercontext.Context{
					Clients: controllercontext.Clients{
						K8s: client,
					},
					Catalog: v1alpha1.Catalog{},
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			err = r.EnsureCreated(ctx, tc.app)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var updated v1alpha1.App
			err = r.ctrlClient.Get(
				ctx,
				types.NamespacedName{Name: tc.app.Name, Namespace: tc.app.Namespace},
				&updated,
			)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if !reflect.DeepEqual(key.AppStatus(updated), tc.expectedStatus) {
				t.Fatalf("want matching statuses \n %s", cmp.Diff(key.AppStatus(updated), tc.expectedStatus))
			}
		})
	}
}
