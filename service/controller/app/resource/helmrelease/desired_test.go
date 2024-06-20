package helmrelease

import (
	"context"
	"reflect"
	"regexp"
	"testing"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache/indexcachetest"
)

func Test_Resource_GetDesiredState(t *testing.T) {
	tests := []struct {
		name                string
		obj                 *v1alpha1.App
		catalog             v1alpha1.Catalog
		configMap           *corev1.ConfigMap
		secret              *corev1.Secret
		index               *indexcache.Index
		expectedHelmRelease *helmv2.HelmRelease
		expectedChartStatus *controllercontext.ChartStatus
		errorPattern        *regexp.Regexp
		error               bool
	}{
		{
			name: "case 0: flawless flow",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.AppSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					Version: "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.io/app-catalog/",
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "https://giantswarm.github.io/app-catalog/",
						},
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-cool-prometheus-helmrelease-values",
					Namespace:       "default",
					ResourceVersion: "1234",
				},
			},
			index: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Annotations: map[string]string{
						annotation.AppOperatorLatestConfigMapVersion: "1234",
						fluxmeta.ReconcileRequestAnnotation:          "1234",
					},
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
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
					ReleaseName:      "my-cool-prometheus",
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Install: &helmv2.Install{
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					Rollback: &helmv2.Rollback{},
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-helmrelease-values",
						},
					},
				},
			},
		},
		{
			name: "case 1: generating catalog url failed",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "kubernetes-prometheus",
					Namespace: "monitoring",
					Version:   "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "", // Empty baseURL
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "", // Empty baseURL
						},
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus-helmrelease-values",
					Namespace: "default",
				},
			},
			expectedHelmRelease: &helmv2.HelmRelease{},
			expectedChartStatus: &controllercontext.ChartStatus{
				Reason: "index not found error: index (*indexcache.Index)(nil) for \"\" is <nil>",
				Status: "index-not-found",
			},
			error: false,
		},
		{
			name: "case 2: flawless flow with prefixed version",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.AppSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					Version: "v1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.io/app-catalog/",
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "https://giantswarm.github.io/app-catalog/",
						},
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-cool-prometheus-helmrelease-values",
					Namespace:       "default",
					ResourceVersion: "1234",
				},
			},
			index: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Annotations: map[string]string{
						annotation.AppOperatorLatestConfigMapVersion: "1234",
						fluxmeta.ReconcileRequestAnnotation:          "1234",
					},
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
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
					ReleaseName:      "my-cool-prometheus",
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Install: &helmv2.Install{
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					Rollback: &helmv2.Rollback{},
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-helmrelease-values",
						},
					},
				},
			},
		},
		{
			name: "case 3: relative URL in index.yaml",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "prometheus",
					Namespace: "monitoring",
					Version:   "v1.0.0",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.io/app-catalog/",
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "https://giantswarm.github.io/app-catalog/",
						},
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			index: newIndexWithApp("prometheus", "1.0.0", "/prometheus-1.0.0.tgz"),
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
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
					ReleaseName:      "my-cool-prometheus",
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Install: &helmv2.Install{
						DisableOpenAPIValidation: true,
						CreateNamespace:          true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					Rollback: &helmv2.Rollback{},
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
				},
			},
		},
		{
			name: "case 4: use custom timeout settings",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "default",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "hello-world-app",
					Namespace: "default",
					Version:   "1.1.1",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
					Install: v1alpha1.AppSpecInstall{
						Timeout: &metav1.Duration{Duration: 360 * time.Second},
					},
					Rollback: v1alpha1.AppSpecRollback{
						Timeout: &metav1.Duration{Duration: 420 * time.Second},
					},
					Uninstall: v1alpha1.AppSpecUninstall{
						Timeout: &metav1.Duration{Duration: 480 * time.Second},
					},
					Upgrade: v1alpha1.AppSpecUpgrade{
						Timeout: &metav1.Duration{Duration: 540 * time.Second},
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.io/app-catalog/",
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "https://giantswarm.github.io/app-catalog/",
						},
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			index: newIndexWithApp("hello-world-app", "1.1.1", "https://giantswarm.github.io/app-catalog/hello-world-app-1.1.1.tgz"),
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "default",
					Labels: map[string]string{
						"giantswarm.io/managed-by": "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "hello-world-app",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.1.1",
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
					ReleaseName:      "hello-world",
					StorageNamespace: "default",
					TargetNamespace:  "default",
					Install: &helmv2.Install{
						CreateNamespace:          true,
						DisableOpenAPIValidation: true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
						Timeout: &metav1.Duration{Duration: 6 * time.Minute},
					},
					Upgrade: &helmv2.Upgrade{
						Timeout: &metav1.Duration{Duration: 9 * time.Minute},
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					Rollback: &helmv2.Rollback{
						Timeout: &metav1.Duration{Duration: 7 * time.Minute},
					},
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
						Timeout:             &metav1.Duration{Duration: 8 * time.Minute},
					},
				},
			},
		},
		{
			name: "case 5: config maps and secrets are only set via extra configs",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "cluster-operator",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "giantswarm",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
					Name:      "prometheus",
					Namespace: "monitoring",
					Version:   "1.0.0",
					ExtraConfigs: []v1alpha1.AppExtraConfig{
						{
							Kind:      "configMap",
							Name:      "configmap-extra-config",
							Namespace: "giantswarm",
						},
						{
							Kind:      "secret",
							Name:      "secret-extra-config",
							Namespace: "giantswarm",
						},
					},
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-cool-prometheus-helmrelease-values",
					Namespace:       "default",
					ResourceVersion: "1234",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-cool-prometheus-helmrelease-secrets",
					Namespace:       "default",
					ResourceVersion: "4321",
				},
				Data: map[string][]byte{
					"values": []byte("Zm9vOiBiYXIK"),
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.io/app-catalog/",
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "https://giantswarm.github.io/app-catalog/",
						},
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			index: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Annotations: map[string]string{
						annotation.AppOperatorLatestConfigMapVersion: "1234",
						annotation.AppOperatorLatestSecretVersion:    "4321",
						fluxmeta.ReconcileRequestAnnotation:          "12344321",
					},
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
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
					ReleaseName:      "my-cool-prometheus",
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Install: &helmv2.Install{
						CreateNamespace:          true,
						DisableOpenAPIValidation: true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					Rollback: &helmv2.Rollback{},
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					ValuesFrom: []helmv2.ValuesReference{
						helmv2.ValuesReference{
							Kind: "ConfigMap",
							Name: "my-cool-prometheus-helmrelease-values",
						},
						helmv2.ValuesReference{
							Kind: "Secret",
							Name: "my-cool-prometheus-helmrelease-secrets",
						},
					},
				},
			},
		},
		{
			name: "case 7: app not found in the catalog",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "missing-app",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "missing-app",
					Namespace: "default",
					Version:   "1.0.0",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "", // Empty baseURL
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "", // Empty baseURL
						},
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			index:               newIndexWithApp("existing-app", "1.0.0", "https://giantswarm.github.io/app-catalog/existing-app-1.0.0.tgz"),
			expectedHelmRelease: &helmv2.HelmRelease{},
			expectedChartStatus: &controllercontext.ChartStatus{
				Reason: "app not found error: no entries for app `missing-app` in index.yaml for \"\"",
				Status: "app-not-found",
			},
			error: false,
		},
		{
			name: "case 8: app version not found in the catalog",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "missing-version-app",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "existing-app",
					Namespace: "default",
					Version:   "2.0.0",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "giantswarm-12345",
							Namespace: "12345",
						},
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "", // Empty baseURL
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "", // Empty baseURL
						},
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			index:               newIndexWithApp("existing-app", "1.0.0", "https://giantswarm.github.io/app-catalog/existing-app-1.0.0.tgz"),
			expectedHelmRelease: &helmv2.HelmRelease{},
			expectedChartStatus: &controllercontext.ChartStatus{
				Reason: "app version not found error: no app `existing-app` in index.yaml with given version `2.0.0`",
				Status: "app-version-not-found",
			},
			error: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0)
			if tc.configMap != nil {
				objs = append(objs, tc.configMap)
			}
			if tc.secret != nil {
				objs = append(objs, tc.secret)
			}

			s := runtime.NewScheme()
			_ = helmv2.AddToScheme(s)
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.AppList{})

			c := Config{
				IndexCache: indexcachetest.New(indexcachetest.Config{
					GetIndexResponse: tc.index,
				}),
				Logger:     microloggertest.New(),
				CtrlClient: fake.NewClientBuilder().WithScheme(s).Build(),

				DependencyWaitTimeoutMinutes: 30,
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var ctx context.Context
			{
				config := k8sclienttest.ClientsConfig{
					CtrlClient: fake.NewClientBuilder().WithScheme(s).Build(),
					K8sClient:  clientgofake.NewSimpleClientset(objs...),
				}
				client := k8sclienttest.NewClients(config)

				c := controllercontext.Context{
					Clients: controllercontext.Clients{
						K8s: client,
					},
					Catalog: tc.catalog,
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			result, err := r.GetDesiredState(ctx, tc.obj)
			switch {
			case err != nil && !tc.error:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.error:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && tc.error && !tc.errorPattern.MatchString(microerror.Pretty(err, true)):
				t.Fatalf("error == %q does not match expected pattern %q", err.Error(), tc.errorPattern.String())
			}

			if err == nil && !tc.error {
				chart, err := toHelmRelease(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if tc.expectedChartStatus != nil {
					cc, err := controllercontext.FromContext(ctx)
					if err != nil {
						t.Fatalf("error == %#v, want nil", err)
					}

					if !reflect.DeepEqual(cc.Status.ChartStatus, *tc.expectedChartStatus) {
						t.Fatalf("want matching statuses \n %s", cmp.Diff(cc.Status.ChartStatus, *tc.expectedChartStatus))
					}
				}

				if !reflect.DeepEqual(chart.ObjectMeta, tc.expectedHelmRelease.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(chart.ObjectMeta, tc.expectedHelmRelease.ObjectMeta))
				}

				if !reflect.DeepEqual(chart.Spec, tc.expectedHelmRelease.Spec) {
					t.Fatalf("want matching spec \n %s", cmp.Diff(chart.Spec, tc.expectedHelmRelease.Spec))
				}

				if !reflect.DeepEqual(chart.TypeMeta, tc.expectedHelmRelease.TypeMeta) {
					t.Fatalf("want matching typemeta \n %s", cmp.Diff(chart.TypeMeta, tc.expectedHelmRelease.TypeMeta))
				}
			}
		})
	}
}

func Test_Resource_Bulid_TarballURL(t *testing.T) {
	now := time.Now()
	app := &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-cool-prometheus",
			Namespace: "default",
			Labels: map[string]string{
				"app":                                "prometheus",
				"app-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":           "cluster-operator",
			},
		},
		Spec: v1alpha1.AppSpec{
			Catalog:   "giantswarm",
			Name:      "prometheus",
			Namespace: "monitoring",
			NamespaceConfig: v1alpha1.AppSpecNamespaceConfig{
				Annotations: map[string]string{
					"linkerd.io/inject": "enabled",
				},
			},
			Version: "1.0.0",
			Config:  v1alpha1.AppSpecConfig{},
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				Secret: v1alpha1.AppSpecKubeConfigSecret{
					Name:      "giantswarm-12345",
					Namespace: "12345",
				},
			},
		},
	}
	internalCatalog := v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "giantswarm",
			Namespace: "default",
			Labels: map[string]string{
				"app-operator.giantswarm.io/version":           "1.0.0",
				"application.giantswarm.io/catalog-visibility": "internal",
			},
		},
		Spec: v1alpha1.CatalogSpec{
			Title:       "Giant Swarm",
			Description: "Catalog of Apps by Giant Swarm",
			Storage: v1alpha1.CatalogSpecStorage{
				Type: "helm",
				URL:  "https://giantswarm.github.io/app-catalog/",
			},
			Repositories: []v1alpha1.CatalogSpecRepository{
				{
					Type: "helm",
					URL:  "https://giantswarm.github.io/app-catalog/",
				},
				{
					Type: "oci",
					URL:  "oci://giantswarmpublic.azurecr.io/app-catalog/",
				},
			},
			LogoURL: "https://s.giantswarm.io/...",
		},
	}
	externalCatalog := v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "giantswarm",
			Namespace: "default",
			Labels: map[string]string{
				"app-operator.giantswarm.io/version": "1.0.0",
			},
		},
		Spec: v1alpha1.CatalogSpec{
			Title:       "Giant Swarm",
			Description: "Catalog of Apps by Giant Swarm",
			Storage: v1alpha1.CatalogSpecStorage{
				Type: "helm",
				URL:  "https://giantswarm.github.io/app-catalog-mirror/",
			},
			Repositories: []v1alpha1.CatalogSpecRepository{
				{
					Type: "helm",
					URL:  "https://giantswarm.github.io/app-catalog-mirror/",
				},
				{
					Type: "helm",
					URL:  "https://giantswarm.github.io/app-catalog-second-mirror/",
				},
				{
					Type: "helm",
					URL:  "https://giantswarm.github.io/app-catalog/",
				},
			},
			LogoURL: "https://s.giantswarm.io/...",
		},
	}

	tests := []struct {
		name                string
		obj                 *v1alpha1.App
		catalog             v1alpha1.Catalog
		indices             map[string]indexcachetest.Config
		existingHelmChart   *sourcev1.HelmChart
		existingHelmRelease *helmv2.HelmRelease
		expectedHelmRelease *helmv2.HelmRelease
		errorPattern        *regexp.Regexp
		error               bool
	}{
		{
			name:    "case 0: HelmRelease CR does not exist yet, pick first repository",
			obj:     app,
			catalog: internalCatalog,
			indices: map[string]indexcachetest.Config{},
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
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
					Install: &helmv2.Install{
						CreateNamespace:          true,
						DisableOpenAPIValidation: true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Rollback:         &helmv2.Rollback{},
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					ReleaseName: "my-cool-prometheus",
				},
			},
		},
		{
			name:    "case 1: HelmRelease CR exists with unknown repository, pick first",
			obj:     app,
			catalog: internalCatalog,
			indices: map[string]indexcachetest.Config{},
			existingHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-helm-THIS.REPO.DOES.NOT.EXIST.IN.CATALOG-app-catalog",
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
					Install: &helmv2.Install{
						CreateNamespace:          true,
						DisableOpenAPIValidation: true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Rollback:         &helmv2.Rollback{},
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					ReleaseName: "my-cool-prometheus",
				},
			},
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
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
					Install: &helmv2.Install{
						CreateNamespace:          true,
						DisableOpenAPIValidation: true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Rollback:         &helmv2.Rollback{},
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					ReleaseName: "my-cool-prometheus",
				},
			},
		},
		{
			name:    "case 2: HelmRelease CR and HelmChart CR both report tarball problems, use next in line repository",
			obj:     app,
			catalog: internalCatalog,
			indices: map[string]indexcachetest.Config{},
			existingHelmChart: &sourcev1.HelmChart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default-my-cool-prometheus",
					Namespace: "default",
				},
				Status: sourcev1.HelmChartStatus{
					Conditions: []metav1.Condition{
						metav1.Condition{
							Type:    sourcev1.StorageOperationFailedCondition,
							Status:  metav1.ConditionTrue,
							Message: "unable to copy Helm chart to storage",
							Reason:  sourcev1.ArchiveOperationFailedReason,
						},
					},
				},
			},
			existingHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
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
					Install: &helmv2.Install{
						CreateNamespace:          true,
						DisableOpenAPIValidation: true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Rollback:         &helmv2.Rollback{},
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					ReleaseName: "my-cool-prometheus",
				},
				Status: helmv2.HelmReleaseStatus{
					Conditions: []metav1.Condition{
						metav1.Condition{
							Reason:             helmv2.UpgradeSucceededReason,
							LastTransitionTime: metav1.NewTime(now.Add(-12 * time.Minute)),
							Type:               helmv2.ReleasedCondition,
						},
						metav1.Condition{
							Reason:             helmv2.ArtifactFailedReason,
							LastTransitionTime: metav1.NewTime(now.Add(-10 * time.Minute)),
							Type:               fluxmeta.ReadyCondition,
						},
						metav1.Condition{
							Reason:             helmv2.InstallSucceededReason,
							LastTransitionTime: metav1.NewTime(now.Add(-15 * time.Minute)),
							Type:               helmv2.ReleasedCondition,
						},
					},
				},
			},
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
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
					Install: &helmv2.Install{
						CreateNamespace:          true,
						DisableOpenAPIValidation: true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Rollback:         &helmv2.Rollback{},
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					ReleaseName: "my-cool-prometheus",
				},
			},
		},
		{
			name:    "case 3: HelmRelease CR reports tarball problems, but HelmChart CR doesn't, re-use old repository",
			obj:     app,
			catalog: internalCatalog,
			indices: map[string]indexcachetest.Config{},
			existingHelmChart: &sourcev1.HelmChart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default-my-cool-prometheus",
					Namespace: "default",
				},
				Status: sourcev1.HelmChartStatus{
					Conditions: []metav1.Condition{},
				},
			},
			existingHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
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
					Install: &helmv2.Install{
						CreateNamespace:          true,
						DisableOpenAPIValidation: true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Rollback:         &helmv2.Rollback{},
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					ReleaseName: "my-cool-prometheus",
				},
				Status: helmv2.HelmReleaseStatus{
					Conditions: []metav1.Condition{
						metav1.Condition{
							Reason:             helmv2.UpgradeSucceededReason,
							LastTransitionTime: metav1.NewTime(now.Add(-12 * time.Minute)),
							Type:               helmv2.ReleasedCondition,
						},
						metav1.Condition{
							Reason:             helmv2.ArtifactFailedReason,
							LastTransitionTime: metav1.NewTime(now.Add(-10 * time.Minute)),
							Type:               fluxmeta.ReadyCondition,
						},
						metav1.Condition{
							Reason:             helmv2.InstallSucceededReason,
							LastTransitionTime: metav1.NewTime(now.Add(-15 * time.Minute)),
							Type:               helmv2.ReleasedCondition,
						},
					},
				},
			},
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
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
					Install: &helmv2.Install{
						CreateNamespace:          true,
						DisableOpenAPIValidation: true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Rollback:         &helmv2.Rollback{},
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					ReleaseName: "my-cool-prometheus",
				},
			},
		},
		{
			name:    "case 4: Walk through fallback repositories until one works",
			obj:     app,
			catalog: externalCatalog,
			indices: map[string]indexcachetest.Config{
				"https://giantswarm.github.io/app-catalog/": {
					GetIndexResponse: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
				},
			},
			expectedHelmRelease: &helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       helmv2.HelmReleaseKind,
					APIVersion: helmv2.GroupVersion.Group,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "default",
					Labels: map[string]string{
						"app":                                "prometheus",
						"app-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":           "app-operator",
					},
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:             "prometheus",
							ReconcileStrategy: "ChartVersion",
							Version:           "1.0.0",
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
					Install: &helmv2.Install{
						CreateNamespace:          true,
						DisableOpenAPIValidation: true,
						Remediation: &helmv2.InstallRemediation{
							Retries: 3,
						},
					},
					Rollback:         &helmv2.Rollback{},
					StorageNamespace: "monitoring",
					TargetNamespace:  "monitoring",
					Uninstall: &helmv2.Uninstall{
						DeletionPropagation: ptr.To("background"),
					},
					Upgrade: &helmv2.Upgrade{
						Remediation: &helmv2.UpgradeRemediation{
							Retries: 3,
						},
					},
					ReleaseName: "my-cool-prometheus",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0)
			if tc.existingHelmRelease != nil {
				objs = append(objs, tc.existingHelmRelease)
			}

			if tc.existingHelmChart != nil {
				objs = append(objs, tc.existingHelmChart)
			}

			s := runtime.NewScheme()
			_ = helmv2.AddToScheme(s)
			_ = sourcev1.AddToScheme(s)

			c := Config{
				IndexCache: indexcachetest.NewMap(tc.indices),
				Logger:     microloggertest.New(),
				CtrlClient: fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build(),

				DependencyWaitTimeoutMinutes: 30,
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			var ctx context.Context
			{
				config := k8sclienttest.ClientsConfig{
					CtrlClient: fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build(),
					K8sClient:  clientgofake.NewSimpleClientset(),
				}
				client := k8sclienttest.NewClients(config)

				c := controllercontext.Context{
					Clients: controllercontext.Clients{
						K8s: client,
					},
					Catalog: tc.catalog,
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			result, err := r.GetDesiredState(ctx, tc.obj)
			switch {
			case err != nil && !tc.error:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.error:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && tc.error && !tc.errorPattern.MatchString(microerror.Pretty(err, true)):
				t.Fatalf("error == %q does not match expected pattern %q", err.Error(), tc.errorPattern.String())
			}

			if err == nil && !tc.error {
				hr, err := toHelmRelease(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(hr.ObjectMeta, tc.expectedHelmRelease.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(hr.ObjectMeta, tc.expectedHelmRelease.ObjectMeta))
				}

				if !reflect.DeepEqual(hr.Spec, tc.expectedHelmRelease.Spec) {
					t.Fatalf("want matching spec \n %s", cmp.Diff(hr.Spec, tc.expectedHelmRelease.Spec))
				}

				if !reflect.DeepEqual(hr.TypeMeta, tc.expectedHelmRelease.TypeMeta) {
					t.Fatalf("want matching typemeta \n %s", cmp.Diff(hr.TypeMeta, tc.expectedHelmRelease.TypeMeta))
				}
			}
		})
	}
}

func Test_generateConfig(t *testing.T) {
	tests := []struct {
		name             string
		cr               v1alpha1.App
		catalog          v1alpha1.Catalog
		secret           *corev1.Secret
		configMap        *corev1.ConfigMap
		expectedConfig   []helmv2.ValuesReference
		expectedRevision map[string]string
	}{
		{
			name:             "case 0: no config",
			cr:               v1alpha1.App{},
			catalog:          v1alpha1.Catalog{},
			expectedConfig:   []helmv2.ValuesReference{},
			expectedRevision: map[string]string{},
		},
		{
			name: "case 1: has a configmap from app",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
					Namespace: "giantswarm",
				},
			},
			catalog: v1alpha1.Catalog{},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-app-helmrelease-values",
					Namespace:       "default",
					ResourceVersion: "1234",
				},
			},
			expectedConfig: []helmv2.ValuesReference{
				helmv2.ValuesReference{
					Kind: "ConfigMap",
					Name: "test-app-helmrelease-values",
				},
			},
			expectedRevision: map[string]string{
				annotation.AppOperatorLatestConfigMapVersion: "1234",
			},
		},
		{
			name: "case 2: has a secret from app",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
					Namespace: "giantswarm",
				},
			},
			catalog: v1alpha1.Catalog{},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-app-helmrelease-secrets",
					Namespace:       "default",
					ResourceVersion: "4321",
				},
			},
			expectedConfig: []helmv2.ValuesReference{
				helmv2.ValuesReference{
					Kind: "Secret",
					Name: "test-app-helmrelease-secrets",
				},
			},
			expectedRevision: map[string]string{
				annotation.AppOperatorLatestSecretVersion: "4321",
			},
		},
		{
			name: "case 3: has both a configmap and secret from app",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "test-app-values",
							Namespace: "default",
						},
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
					Namespace: "giantswarm",
				},
			},
			catalog: v1alpha1.Catalog{},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-app-helmrelease-values",
					Namespace:       "default",
					ResourceVersion: "1234",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-app-helmrelease-secrets",
					Namespace:       "default",
					ResourceVersion: "4321",
				},
			},
			expectedConfig: []helmv2.ValuesReference{
				helmv2.ValuesReference{
					Kind: "ConfigMap",
					Name: "test-app-helmrelease-values",
				},
				helmv2.ValuesReference{
					Kind: "Secret",
					Name: "test-app-helmrelease-secrets",
				},
			},
			expectedRevision: map[string]string{
				annotation.AppOperatorLatestConfigMapVersion: "1234",
				annotation.AppOperatorLatestSecretVersion:    "4321",
			},
		},
		{
			name: "case 4: has a configmap from catalog",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			catalog: v1alpha1.Catalog{
				Spec: v1alpha1.CatalogSpec{
					Config: &v1alpha1.CatalogSpecConfig{
						ConfigMap: &v1alpha1.CatalogSpecConfigConfigMap{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-app-helmrelease-values",
					Namespace:       "default",
					ResourceVersion: "1234",
				},
			},
			expectedConfig: []helmv2.ValuesReference{
				helmv2.ValuesReference{
					Kind: "ConfigMap",
					Name: "test-app-helmrelease-values",
				},
			},
			expectedRevision: map[string]string{
				annotation.AppOperatorLatestConfigMapVersion: "1234",
			},
		},
		{
			name: "case 5: has a secret from catalog",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			catalog: v1alpha1.Catalog{
				Spec: v1alpha1.CatalogSpec{
					Config: &v1alpha1.CatalogSpecConfig{
						Secret: &v1alpha1.CatalogSpecConfigSecret{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-app-helmrelease-secrets",
					Namespace:       "default",
					ResourceVersion: "4321",
				},
			},
			expectedConfig: []helmv2.ValuesReference{
				helmv2.ValuesReference{
					Kind: "Secret",
					Name: "test-app-helmrelease-secrets",
				},
			},
			expectedRevision: map[string]string{
				annotation.AppOperatorLatestSecretVersion: "4321",
			},
		},
		{
			name: "case 6: has both a configmap and secret from catalog",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			catalog: v1alpha1.Catalog{
				Spec: v1alpha1.CatalogSpec{
					Config: &v1alpha1.CatalogSpecConfig{
						ConfigMap: &v1alpha1.CatalogSpecConfigConfigMap{
							Name:      "test-app-values",
							Namespace: "default",
						},
						Secret: &v1alpha1.CatalogSpecConfigSecret{
							Name:      "test-app-values",
							Namespace: "default",
						},
					},
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-app-helmrelease-values",
					Namespace:       "default",
					ResourceVersion: "1234",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-app-helmrelease-secrets",
					Namespace:       "default",
					ResourceVersion: "4321",
				},
			},
			expectedConfig: []helmv2.ValuesReference{
				helmv2.ValuesReference{
					Kind: "ConfigMap",
					Name: "test-app-helmrelease-values",
				},
				helmv2.ValuesReference{
					Kind: "Secret",
					Name: "test-app-helmrelease-secrets",
				},
			},
			expectedRevision: map[string]string{
				annotation.AppOperatorLatestConfigMapVersion: "1234",
				annotation.AppOperatorLatestSecretVersion:    "4321",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0)
			if tc.configMap != nil {
				objs = append(objs, tc.configMap)
			}

			if tc.secret != nil {
				objs = append(objs, tc.secret)
			}

			client := clientgofake.NewSimpleClientset(objs...)

			result, version, err := generateConfig(context.Background(), client, tc.cr, tc.catalog)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if !reflect.DeepEqual(result, tc.expectedConfig) {
				t.Fatalf("want matching Config \n %s", cmp.Diff(result, tc.expectedConfig))
			}

			if !reflect.DeepEqual(version, tc.expectedRevision) {
				t.Fatalf("want matching Revision \n %s", cmp.Diff(version, tc.expectedRevision))
			}
		})
	}
}

func Test_processLabels(t *testing.T) {
	tests := []struct {
		name           string
		projectName    string
		inputLabels    map[string]string
		expectedLabels map[string]string
	}{
		{
			name:        "case 0: basic match",
			projectName: "app-operator",
			inputLabels: map[string]string{
				"app-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":           "release-operator",
			},
			expectedLabels: map[string]string{
				"app-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":           "app-operator",
			},
		},
		{
			name:        "case 1: extra labels still present",
			projectName: "app-operator",
			inputLabels: map[string]string{
				"app":                                "prometheus",
				"app-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/cluster":              "5xchu",
				"giantswarm.io/managed-by":           "cluster-operator",
				"giantswarm.io/organization":         "giantswarm",
			},
			expectedLabels: map[string]string{
				"app":                                "prometheus",
				"app-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/cluster":              "5xchu",
				"giantswarm.io/managed-by":           "app-operator",
				"giantswarm.io/organization":         "giantswarm",
			},
		},
		{
			name:        "case 2: empty inputs",
			projectName: "app-operator",
			expectedLabels: map[string]string{
				"giantswarm.io/managed-by": "app-operator",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			result := processLabels(tc.projectName, tc.inputLabels)

			if !reflect.DeepEqual(result, tc.expectedLabels) {
				t.Fatalf("want matching \n %s", cmp.Diff(result, tc.expectedLabels))
			}
		})
	}
}

func Test_getDependenciesFromCR(t *testing.T) {
	tests := []struct {
		name            string
		annotationValue string
		want            []string
		wantErr         bool
	}{
		{
			name:            "Annotation exists with one app",
			annotationValue: "coredns",
			want:            []string{"coredns"},
			wantErr:         false,
		},
		{
			name:            "Annotation exists with two apps",
			annotationValue: "coredns,prometheus",
			want:            []string{"coredns", "prometheus"},
			wantErr:         false,
		},
		{
			name:            "Annotation does not exist",
			annotationValue: "",
			want:            []string{},
			wantErr:         false,
		},
		{
			name:            "Annotation with typo",
			annotationValue: "coredns,,prometheus",
			want:            []string{"coredns", "prometheus"},
			wantErr:         false,
		},
		{
			name:            "Annotation with just a comma",
			annotationValue: ",",
			want:            []string{},
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := map[string]string{}
			if tt.annotationValue != "" {
				annotations["app-operator.giantswarm.io/depends-on"] = tt.annotationValue
			}
			app := v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
				},
			}
			got, err := getDependenciesFromCR(app)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDependenciesFromCR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDependenciesFromCR() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func newIndexWithApp(app, version, url string) *indexcache.Index {
	index := &indexcache.Index{
		Entries: map[string][]indexcache.Entry{
			app: {
				{
					Urls: []string{
						url,
					},
					Version: version,
				},
			},
		},
	}

	return index
}
