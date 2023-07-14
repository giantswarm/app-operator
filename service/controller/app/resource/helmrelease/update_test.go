package helmrelease

import (
	"context"
	"reflect"
	"testing"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache/indexcachetest"
)

func Test_Resource_newUpdateChange(t *testing.T) {
	tests := []struct {
		name                string
		currentHelmRelease  *helmv2.HelmRelease
		desiredHelmRelease  *helmv2.HelmRelease
		expectedHelmRelease *helmv2.HelmRelease
		error               bool
	}{
		{
			name: "case 0: flawless flow",
			currentHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-cool-prometheus",
					Namespace:       "giantswarm",
					ResourceVersion: "12345",
					UID:             "51eeec1d-3716-4006-92b4-e7e99f8ab311",
					Annotations: map[string]string{
						"giantswarm.io/sample": "it should be deleted",
					},
					Labels: map[string]string{
						"giantswarm.io/managed-by": "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "prometheus",
							Version: "0.0.9",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "my-cool-prometheus",
					TargetNamespace: "monitoring",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-chart-values",
						},
					},
				},
			},
			desiredHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
					Annotations: map[string]string{},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "prometheus",
							Version: "1.0.0",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "my-cool-prometheus",
					TargetNamespace: "monitoring",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-chart-values",
						},
					},
				},
			},
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-cool-prometheus",
					Namespace:       "giantswarm",
					ResourceVersion: "12345",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
					Annotations: map[string]string{},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "prometheus",
							Version: "1.0.0",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "my-cool-prometheus",
					TargetNamespace: "monitoring",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-chart-values",
						},
					},
				},
			},
		},
		{
			name: "case 1: same chart",
			currentHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
					Annotations: map[string]string{},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "prometheus",
							Version: "1.0.0",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "my-cool-prometheus",
					TargetNamespace: "monitoring",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-chart-values",
						},
					},
				},
			},
			desiredHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
					Annotations: map[string]string{},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "prometheus",
							Version: "1.0.0",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "my-cool-prometheus",
					TargetNamespace: "monitoring",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-chart-values",
						},
					},
				},
			},
			expectedHelmRelease: &helmv2.HelmRelease{},
		},
		{
			name: "case 2: adding timeout",
			currentHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "default",
					Labels: map[string]string{
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "hello-world-app",
							Version: "1.1.1",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "hello-world",
					TargetNamespace: "default",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{},
				},
			},
			desiredHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "default",
					Labels: map[string]string{
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "hello-world-app",
							Version: "1.1.1",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "hello-world",
					TargetNamespace: "default",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
						Timeout:                  &metav1.Duration{Duration: 300 * time.Second},
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{},
				},
			},
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "default",
					Labels: map[string]string{
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "hello-world-app",
							Version: "1.1.1",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "hello-world",
					TargetNamespace: "default",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
						Timeout:                  &metav1.Duration{Duration: 300 * time.Second},
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{},
				},
			},
		},
		{
			name: "case 3: updating values version",
			currentHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-cool-prometheus",
					Namespace:       "giantswarm",
					ResourceVersion: "12345",
					UID:             "51eeec1d-3716-4006-92b4-e7e99f8ab311",
					Annotations: map[string]string{
						annotation.AppOperatorLatestConfigMapVersion: "1",
					},
					Labels: map[string]string{
						"giantswarm.io/managed-by": "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "prometheus",
							Version: "0.0.9",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "my-cool-prometheus",
					TargetNamespace: "monitoring",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-chart-values",
						},
					},
				},
			},
			desiredHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
					Annotations: map[string]string{
						annotation.AppOperatorLatestConfigMapVersion: "2",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "prometheus",
							Version: "1.0.0",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "my-cool-prometheus",
					TargetNamespace: "monitoring",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-chart-values",
						},
					},
				},
			},
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-cool-prometheus",
					Namespace:       "giantswarm",
					ResourceVersion: "12345",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
					Annotations: map[string]string{
						annotation.AppOperatorLatestConfigMapVersion: "2",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "prometheus",
							Version: "1.0.0",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
						},
					},
					Interval: metav1.Duration{Duration: 10 * time.Minute},
					KubeConfig: &fluxmeta.KubeConfigReference{
						SecretRef: fluxmeta.SecretKeyReference{
							Name: "giantswarm-12345",
						},
					},
					ReleaseName:     "my-cool-prometheus",
					TargetNamespace: "monitoring",
					Install: &helmv2.Install{
						DisableWait:              true,
						DisableWaitForJobs:       true,
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
					},
					Upgrade: &helmv2.Upgrade{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Rollback: &helmv2.Rollback{
						DisableWait:        true,
						DisableWaitForJobs: true,
					},
					Uninstall: &helmv2.Uninstall{
						DisableWait: true,
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-chart-values",
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := Config{
				IndexCache: indexcachetest.New(indexcachetest.Config{}),
				Logger:     microloggertest.New(),
				CtrlClient: fake.NewFakeClient(), //nolint:staticcheck

				DependencyWaitTimeoutMinutes: 30,
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var ctx context.Context
			{
				config := k8sclienttest.ClientsConfig{
					CtrlClient: fake.NewFakeClient(), //nolint:staticcheck
					K8sClient:  clientgofake.NewSimpleClientset(),
				}
				client := k8sclienttest.NewClients(config)

				c := controllercontext.Context{
					Clients: controllercontext.Clients{
						K8s: client,
					},
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			result, err := r.newUpdateChange(ctx, tc.currentHelmRelease, tc.desiredHelmRelease)
			switch {
			case err != nil && !tc.error:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.error:
				t.Fatalf("error == nil, want non-nil")
			}

			if err == nil && !tc.error {
				helmRelease, err := toHelmRelease(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(helmRelease.ObjectMeta, tc.expectedHelmRelease.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(helmRelease.ObjectMeta, tc.expectedHelmRelease.ObjectMeta))
				}

				if !reflect.DeepEqual(helmRelease.Spec, tc.expectedHelmRelease.Spec) {
					t.Fatalf("want matching spec \n %s", cmp.Diff(helmRelease.Spec, tc.expectedHelmRelease.Spec))
				}

				if !reflect.DeepEqual(helmRelease.TypeMeta, tc.expectedHelmRelease.TypeMeta) {
					t.Fatalf("want matching typemeta \n %s", cmp.Diff(helmRelease.TypeMeta, tc.expectedHelmRelease.TypeMeta))
				}
			}
		})
	}
}
