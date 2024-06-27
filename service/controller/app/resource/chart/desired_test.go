package chart

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache/indexcachetest"
)

func Test_Resource_GetDesiredState(t *testing.T) {
	tm := metav1.NewTime(time.Now())
	tests := []struct {
		name                string
		obj                 *v1alpha1.App
		catalog             v1alpha1.Catalog
		configMap           *corev1.ConfigMap
		secret              *corev1.Secret
		index               *indexcache.Index
		expectedChart       *v1alpha1.Chart
		expectedChartStatus *controllercontext.ChartStatus
		errorPattern        *regexp.Regexp
		error               bool
		workloadClusterId   string
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
					Name:      "my-cool-prometheus-chart-values",
					Namespace: "giantswarm",
				},
			},
			index: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "my-cool-prometheus-chart-values",
							Namespace: "giantswarm",
						},
					},
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					TarballURL: "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
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
					Name:      "my-cool-prometheus-chart-values",
					Namespace: "giantswarm",
				},
			},
			expectedChart: &v1alpha1.Chart{},
			expectedChartStatus: &controllercontext.ChartStatus{
				Reason: "index not found error: index (*indexcache.Index)(nil) for \"\" is <nil>",
				Status: "index-not-found",
			},
			error: false,
		},
		{
			name: "case 2: set helm force upgrade annotation",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/force-helm-upgrade": "true",
					},
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
					Version:   "1.0.0",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						InCluster: true,
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "giantswarm",
				},
				Spec: v1alpha1.CatalogSpec{
					Title: "Giant Swarm",
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
				},
			},
			index: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":           "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace":      "default",
						"chart-operator.giantswarm.io/force-helm-upgrade": "true",
					},
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
		},
		{
			name: "case 3: flawless flow with prefixed version",
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
					Name:      "my-cool-prometheus-chart-values",
					Namespace: "giantswarm",
				},
			},
			index: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "my-cool-prometheus-chart-values",
							Namespace: "giantswarm",
						},
					},
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					TarballURL: "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
		},
		{
			name: "case 4: relative URL in index.yaml",
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
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
		},
		{
			name: "case 5: use custom timeout settings",
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
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "hello-world",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Name:       "hello-world",
					Namespace:  "default",
					TarballURL: "https://giantswarm.github.io/app-catalog/hello-world-app-1.1.1.tgz",
					Version:    "1.1.1",
					Install: v1alpha1.ChartSpecInstall{
						Timeout: &metav1.Duration{Duration: 360 * time.Second},
					},
					Rollback: v1alpha1.ChartSpecRollback{
						Timeout: &metav1.Duration{Duration: 420 * time.Second},
					},
					Uninstall: v1alpha1.ChartSpecUninstall{
						Timeout: &metav1.Duration{Duration: 480 * time.Second},
					},
					Upgrade: v1alpha1.ChartSpecUpgrade{
						Timeout: &metav1.Duration{Duration: 540 * time.Second},
					},
				},
			},
		},
		{
			name: "case 6: config maps and secrets are only set via extra configs",
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
					Name:      "my-cool-prometheus-chart-values",
					Namespace: "giantswarm",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus-chart-secrets",
					Namespace: "giantswarm",
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
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config: v1alpha1.ChartSpecConfig{
						ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
							Name:      "my-cool-prometheus-chart-values",
							Namespace: "giantswarm",
						},
						Secret: v1alpha1.ChartSpecConfigSecret{
							Name:      "my-cool-prometheus-chart-secrets",
							Namespace: "giantswarm",
						},
					},
					Name:       "my-cool-prometheus",
					Namespace:  "monitoring",
					TarballURL: "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
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
			index:         newIndexWithApp("existing-app", "1.0.0", "https://giantswarm.github.io/app-catalog/existing-app-1.0.0.tgz"),
			expectedChart: &v1alpha1.Chart{},
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
			index:         newIndexWithApp("existing-app", "1.0.0", "https://giantswarm.github.io/app-catalog/existing-app-1.0.0.tgz"),
			expectedChart: &v1alpha1.Chart{},
			expectedChartStatus: &controllercontext.ChartStatus{
				Reason: "app version not found error: no app `existing-app` in index.yaml with given version `2.0.0`",
				Status: "app-version-not-found",
			},
			error: false,
		},
		{
			name: "case 9: deleting CAPI workload cluster app",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo01-hello-world",
					Namespace: "org-demo",
					Labels: map[string]string{
						"giantswarm.io/cluster": "demo01",
					},
					DeletionTimestamp: &tm,
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "hello-world",
					Namespace: "default",
					Version:   "1.0.0",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "demo01-kubeconfig",
							Namespace: "org-demo",
						},
					},
				},
			},
			expectedChart: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello-world",
					Namespace: "giantswarm",
				},
			},
			workloadClusterId: "demo01",
		},
		{
			name: "case 10: deleting CAPI management cluster app",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo01-security-bundle",
					Namespace: "org-demo",
					Labels: map[string]string{
						"giantswarm.io/cluster": "demo01",
					},
					DeletionTimestamp: &tm,
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "security-bundle",
					Namespace: "security",
					Version:   "1.0.0",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						InCluster: true,
					},
				},
			},
			expectedChart: &v1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo01-security-bundle",
					Namespace: "giantswarm",
				},
			},
			workloadClusterId: "demo01",
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
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.AppList{})

			c := Config{
				IndexCache: indexcachetest.New(indexcachetest.Config{
					GetIndexResponse: tc.index,
				}),
				Logger:        microloggertest.New(),
				CtrlClient:    fake.NewClientBuilder().WithScheme(s).Build(),
				DynamicClient: dynamicfake.NewSimpleDynamicClient(s),

				ChartNamespace:               "giantswarm",
				DependencyWaitTimeoutMinutes: 30,
			}

			if tc.workloadClusterId != "" {
				c.WorkloadClusterID = tc.workloadClusterId
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
				chart, err := toChart(result)
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

				if !reflect.DeepEqual(chart.ObjectMeta, tc.expectedChart.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(chart.ObjectMeta, tc.expectedChart.ObjectMeta))
				}

				if !reflect.DeepEqual(chart.Spec, tc.expectedChart.Spec) {
					t.Fatalf("want matching spec \n %s", cmp.Diff(chart.Spec, tc.expectedChart.Spec))
				}

				if !reflect.DeepEqual(chart.TypeMeta, tc.expectedChart.TypeMeta) {
					t.Fatalf("want matching typemeta \n %s", cmp.Diff(chart.TypeMeta, tc.expectedChart.TypeMeta))
				}
			}
		})
	}
}

func Test_Resource_Bulid_TarballURL(t *testing.T) {
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
		name          string
		obj           *v1alpha1.App
		catalog       v1alpha1.Catalog
		indices       map[string]indexcachetest.Config
		existingChart *v1alpha1.Chart
		expectedChart *v1alpha1.Chart
		errorPattern  *regexp.Regexp
		error         bool
	}{
		{
			name:    "case 0: [internal] chart does not exist yet, pick first repository",
			obj:     app,
			catalog: internalCatalog,
			indices: map[string]indexcachetest.Config{},
			// index: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config:    v1alpha1.ChartSpecConfig{},
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					TarballURL: "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
		},
		{
			name:    "case 1: [internal] chart exists with unknown repository, pick first",
			obj:     app,
			catalog: internalCatalog,
			indices: map[string]indexcachetest.Config{},
			// index: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
			existingChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config:    v1alpha1.ChartSpecConfig{},
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					TarballURL: "https://THIS.REPO.DOES.NOT.EXIST.IN.CATALOG/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config:    v1alpha1.ChartSpecConfig{},
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					TarballURL: "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
		},
		{
			name:    "case 2: [internal] chart exists with a known repository but chart pull failed, pick next",
			obj:     app,
			catalog: internalCatalog,
			indices: map[string]indexcachetest.Config{},
			// index: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
			existingChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config:    v1alpha1.ChartSpecConfig{},
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					TarballURL: "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
				Status: v1alpha1.ChartStatus{
					AppVersion: "1.0.0",
					Reason:     "Could not pull chart",
					Release: v1alpha1.ChartStatusRelease{
						LastDeployed: nil,
						Revision:     nil,
						Status:       "chart-pull-failed",
					},
					Version: "1.0.0",
				},
			},
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config:    v1alpha1.ChartSpecConfig{},
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					TarballURL: "oci://giantswarmpublic.azurecr.io/app-catalog/prometheus:1.0.0",
					Version:    "1.0.0",
				},
			},
		},
		{
			name:    "case 3: [internal] chart exists with a known repository but chart pull failed, pick next (array boundaries)",
			obj:     app,
			catalog: internalCatalog,
			indices: map[string]indexcachetest.Config{},
			// index: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
			existingChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config:    v1alpha1.ChartSpecConfig{},
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					TarballURL: "oci://giantswarmpublic.azurecr.io/app-catalog/prometheus:1.0.0",
					Version:    "1.0.0",
				},
				Status: v1alpha1.ChartStatus{
					AppVersion: "1.0.0",
					Reason:     "Could not pull chart",
					Release: v1alpha1.ChartStatusRelease{
						LastDeployed: nil,
						Revision:     nil,
						Status:       "chart-pull-failed",
					},
					Version: "1.0.0",
				},
			},
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config:    v1alpha1.ChartSpecConfig{},
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					TarballURL: "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
		},
		{
			name:    "case 4: [external] walk through fallback repositories until one works",
			obj:     app,
			catalog: externalCatalog,
			indices: map[string]indexcachetest.Config{
				"https://giantswarm.github.io/app-catalog/": {
					GetIndexResponse: newIndexWithApp("prometheus", "1.0.0", "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz"),
				},
			},
			expectedChart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-cool-prometheus",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						"chart-operator.giantswarm.io/app-name":      "my-cool-prometheus",
						"chart-operator.giantswarm.io/app-namespace": "default",
					},
					Labels: map[string]string{
						"app":                                  "prometheus",
						"chart-operator.giantswarm.io/version": "1.0.0",
						"giantswarm.io/managed-by":             "app-operator",
					},
				},
				Spec: v1alpha1.ChartSpec{
					Config:    v1alpha1.ChartSpecConfig{},
					Name:      "my-cool-prometheus",
					Namespace: "monitoring",
					NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
						Annotations: map[string]string{
							"linkerd.io/inject": "enabled",
						},
					},
					TarballURL: "https://giantswarm.github.io/app-catalog/prometheus-1.0.0.tgz",
					Version:    "1.0.0",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0)
			if tc.existingChart != nil {
				objs = append(objs, tc.existingChart)
			}

			s := runtime.NewScheme()
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.AppList{})

			c := Config{
				IndexCache:    indexcachetest.NewMap(tc.indices),
				Logger:        microloggertest.New(),
				CtrlClient:    fake.NewClientBuilder().WithScheme(s).Build(),
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
				chart, err := toChart(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(chart.ObjectMeta, tc.expectedChart.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(chart.ObjectMeta, tc.expectedChart.ObjectMeta))
				}

				if !reflect.DeepEqual(chart.Spec, tc.expectedChart.Spec) {
					t.Fatalf("want matching spec \n %s", cmp.Diff(chart.Spec, tc.expectedChart.Spec))
				}

				if !reflect.DeepEqual(chart.TypeMeta, tc.expectedChart.TypeMeta) {
					t.Fatalf("want matching typemeta \n %s", cmp.Diff(chart.TypeMeta, tc.expectedChart.TypeMeta))
				}
			}
		})
	}
}

func Test_generateConfig(t *testing.T) {
	tests := []struct {
		name           string
		cr             v1alpha1.App
		catalog        v1alpha1.Catalog
		secret         *corev1.Secret
		configMap      *corev1.ConfigMap
		expectedConfig v1alpha1.ChartSpecConfig
	}{
		{
			name:           "case 0: no config",
			cr:             v1alpha1.App{},
			catalog:        v1alpha1.Catalog{},
			expectedConfig: v1alpha1.ChartSpecConfig{},
		},
		{
			name: "case 1: has a configmap from app",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
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
					Name:            "test-app-chart-values",
					Namespace:       "giantswarm",
					ResourceVersion: "1234",
				},
			},
			expectedConfig: v1alpha1.ChartSpecConfig{
				ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
					Name:            "test-app-chart-values",
					Namespace:       "giantswarm",
					ResourceVersion: "1234",
				},
			},
		},
		{
			name: "case 2: has a secret from app",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
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
					Name:            "test-app-chart-secrets",
					Namespace:       "giantswarm",
					ResourceVersion: "4321",
				},
			},
			expectedConfig: v1alpha1.ChartSpecConfig{
				Secret: v1alpha1.ChartSpecConfigSecret{
					Name:            "test-app-chart-secrets",
					Namespace:       "giantswarm",
					ResourceVersion: "4321",
				},
			},
		},
		{
			name: "case 3: has both a configmap and secret from app",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
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
					Name:            "test-app-chart-values",
					Namespace:       "giantswarm",
					ResourceVersion: "1234",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-app-chart-secrets",
					Namespace:       "giantswarm",
					ResourceVersion: "4321",
				},
			},
			expectedConfig: v1alpha1.ChartSpecConfig{
				ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
					Name:            "test-app-chart-values",
					Namespace:       "giantswarm",
					ResourceVersion: "1234",
				},
				Secret: v1alpha1.ChartSpecConfigSecret{
					Name:            "test-app-chart-secrets",
					Namespace:       "giantswarm",
					ResourceVersion: "4321",
				},
			},
		},
		{
			name: "case 4: has a configmap from catalog",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
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
					Name:            "test-app-chart-values",
					Namespace:       "giantswarm",
					ResourceVersion: "1234",
				},
			},
			expectedConfig: v1alpha1.ChartSpecConfig{
				ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
					Name:            "test-app-chart-values",
					Namespace:       "giantswarm",
					ResourceVersion: "1234",
				},
			},
		},
		{
			name: "case 5: has a secret from catalog",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
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
					Name:            "test-app-chart-secrets",
					Namespace:       "giantswarm",
					ResourceVersion: "4321",
				},
			},
			expectedConfig: v1alpha1.ChartSpecConfig{
				Secret: v1alpha1.ChartSpecConfigSecret{
					Name:            "test-app-chart-secrets",
					Namespace:       "giantswarm",
					ResourceVersion: "4321",
				},
			},
		},
		{
			name: "case 6: has both a configmap and secret from catalog",
			cr: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
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
					Name:            "test-app-chart-values",
					Namespace:       "giantswarm",
					ResourceVersion: "1234",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-app-chart-secrets",
					Namespace:       "giantswarm",
					ResourceVersion: "4321",
				},
			},
			expectedConfig: v1alpha1.ChartSpecConfig{
				ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
					Name:            "test-app-chart-values",
					Namespace:       "giantswarm",
					ResourceVersion: "1234",
				},
				Secret: v1alpha1.ChartSpecConfigSecret{
					Name:            "test-app-chart-secrets",
					Namespace:       "giantswarm",
					ResourceVersion: "4321",
				},
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

			result, err := generateConfig(context.Background(), client, tc.cr, tc.catalog, "giantswarm")
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			if !reflect.DeepEqual(result, tc.expectedConfig) {
				t.Fatalf("want matching Config \n %s", cmp.Diff(result, tc.expectedConfig))
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
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":             "app-operator",
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
				"app":                                  "prometheus",
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/cluster":                "5xchu",
				"giantswarm.io/managed-by":             "app-operator",
				"giantswarm.io/organization":           "giantswarm",
			},
		},
		{
			name:        "case 2: empty inputs",
			projectName: "app-operator",
			expectedLabels: map[string]string{
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":             "app-operator",
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

func Test_checkDependencies(t *testing.T) {
	tests := []struct {
		name                     string
		appToInstall             *v1alpha1.App
		installedApps            []*v1alpha1.App
		installedHelmReleases    []*unstructured.Unstructured
		dependenciesNotInstalled []string
	}{
		{
			name: " case 0: App does not have dependencies, no apps installed",
			appToInstall: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-case-0",
					Namespace: "org-giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
		},
		{
			name: " case 1: App does not have dependencies, other apps installed",
			appToInstall: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-case-1",
					Namespace: "org-giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			installedApps: []*v1alpha1.App{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-1",
						Namespace: "org-giantswarm",
					},
					Spec: v1alpha1.AppSpec{
						Namespace: "giantswarm",
					},
				},
			},
		},
		{
			name: " case 2: App has 1 dependency, dependency is installed",
			appToInstall: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-case-2",
					Namespace: "org-giantswarm",
					Annotations: map[string]string{
						"app-operator.giantswarm.io/depends-on": "test-app-1",
					},
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			installedApps: []*v1alpha1.App{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-1",
						Namespace: "org-giantswarm",
					},
					Spec: v1alpha1.AppSpec{
						Namespace: "giantswarm",
						Version:   "1.0.0",
					},
					Status: v1alpha1.AppStatus{
						Version: "1.0.0",
						Release: v1alpha1.AppStatusRelease{
							Status: "deployed",
						},
					},
				},
			},
		},
		{
			name: " case 3: App has 2 dependencies, 1 dependency is installed, 1 is not",
			appToInstall: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-case-3",
					Namespace: "org-giantswarm",
					Annotations: map[string]string{
						"app-operator.giantswarm.io/depends-on": "test-app-1,test-app-2",
					},
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			installedApps: []*v1alpha1.App{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-1",
						Namespace: "org-giantswarm",
					},
					Spec: v1alpha1.AppSpec{
						Namespace: "giantswarm",
						Version:   "1.0.0",
					},
					Status: v1alpha1.AppStatus{
						Version: "1.0.0",
						Release: v1alpha1.AppStatusRelease{
							Status: "deployed",
						},
					},
				},
			},
			dependenciesNotInstalled: []string{
				"test-app-2",
			},
		},
		{
			name: " case 4: App has 2 dependencies, both are installed",
			appToInstall: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-case-4",
					Namespace: "org-giantswarm",
					Annotations: map[string]string{
						"app-operator.giantswarm.io/depends-on": "test-app-1,test-app-2",
					},
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			installedApps: []*v1alpha1.App{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-1",
						Namespace: "org-giantswarm",
					},
					Spec: v1alpha1.AppSpec{
						Namespace: "giantswarm",
						Version:   "1.0.0",
					},
					Status: v1alpha1.AppStatus{
						Version: "1.0.0",
						Release: v1alpha1.AppStatusRelease{
							Status: "deployed",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-2",
						Namespace: "org-giantswarm",
					},
					Spec: v1alpha1.AppSpec{
						Namespace: "giantswarm",
						Version:   "2.0.0",
					},
					Status: v1alpha1.AppStatus{
						Version: "2.0.0",
						Release: v1alpha1.AppStatusRelease{
							Status: "deployed",
						},
					},
				},
			},
			dependenciesNotInstalled: []string{},
		},
		{
			name: " case 5: App has 1 HelmRelease dependency, dependency is installed",
			appToInstall: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-case-5",
					Namespace: "org-giantswarm",
					Annotations: map[string]string{
						annotationChartOperatorDependsOn:            "test-app-1",
						annotationChartOperatorDependsOnHelmRelease: "true",
					},
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			installedApps: []*v1alpha1.App{},
			installedHelmReleases: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "helm.toolkit.fluxcd.io/v2beta1",
						"kind":       "HelmRelease",
						"metadata": map[string]interface{}{
							"name":      "test-app-1",
							"namespace": "org-giantswarm",
						},
						"spec": map[string]interface{}{
							"chart": map[string]interface{}{
								"spec": map[string]interface{}{
									"version": "1.0.0",
								},
							},
						},
						"status": map[string]interface{}{
							"lastAppliedRevision": "1.0.0",
							"conditions": []interface{}{
								map[string]interface{}{
									"status": "True",
									"type":   "Ready",
								},
							},
						},
					},
				},
			},
		},
		{
			name: " case 6: App has 1 HelmRelease dependency, dependency is not installed",
			appToInstall: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-case-6",
					Namespace: "org-giantswarm",
					Annotations: map[string]string{
						annotationChartOperatorDependsOn:            "test-app-1",
						annotationChartOperatorDependsOnHelmRelease: "true",
					},
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			installedApps:            []*v1alpha1.App{},
			installedHelmReleases:    []*unstructured.Unstructured{},
			dependenciesNotInstalled: []string{"test-app-1"},
		},
		{
			name: " case 7: App has 1 HelmRelease dependency, dependency is applied but not installed",
			appToInstall: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-case-6",
					Namespace: "org-giantswarm",
					Annotations: map[string]string{
						annotationChartOperatorDependsOn:            "test-app-1",
						annotationChartOperatorDependsOnHelmRelease: "true",
					},
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			installedApps: []*v1alpha1.App{},
			installedHelmReleases: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "helm.toolkit.fluxcd.io/v2beta1",
						"kind":       "HelmRelease",
						"metadata": map[string]interface{}{
							"name":      "test-app-1",
							"namespace": "org-giantswarm",
						},
						"spec": map[string]interface{}{
							"chart": map[string]interface{}{
								"spec": map[string]interface{}{
									"version": "1.0.0",
								},
							},
						},
					},
				},
			},
			dependenciesNotInstalled: []string{"test-app-1"},
		},
		{
			name: " case 8: App has 1 HelmRelease dependency and 1 App dependency, both dependencies are installed",
			appToInstall: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-case-7",
					Namespace: "org-giantswarm",
					Annotations: map[string]string{
						annotationChartOperatorDependsOn:            "test-app-1,test-app-2",
						annotationChartOperatorDependsOnHelmRelease: "true",
					},
				},
				Spec: v1alpha1.AppSpec{
					Namespace: "giantswarm",
				},
			},
			installedApps: []*v1alpha1.App{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-2",
						Namespace: "org-giantswarm",
					},
					Spec: v1alpha1.AppSpec{
						Namespace: "giantswarm",
						Version:   "2.0.0",
					},
					Status: v1alpha1.AppStatus{
						Version: "2.0.0",
						Release: v1alpha1.AppStatusRelease{
							Status: "deployed",
						},
					},
				},
			},
			installedHelmReleases: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "helm.toolkit.fluxcd.io/v2beta1",
						"kind":       "HelmRelease",
						"metadata": map[string]interface{}{
							"name":      "test-app-1",
							"namespace": "org-giantswarm",
						},
						"spec": map[string]interface{}{
							"chart": map[string]interface{}{
								"spec": map[string]interface{}{
									"version": "1.0.0",
								},
							},
						},
						"status": map[string]interface{}{
							"lastAppliedRevision": "1.0.0",
							"conditions": []interface{}{
								map[string]interface{}{
									"status": "True",
									"type":   "Ready",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0)
			objs = append(objs, tc.appToInstall)
			for i := range tc.installedApps {
				objs = append(objs, tc.installedApps[i])
			}
			unstructuredObjs := make([]runtime.Object, 0)
			for i := range tc.installedHelmReleases {
				unstructuredObjs = append(unstructuredObjs, tc.installedHelmReleases[i])
			}

			s := runtime.NewScheme()
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.App{})
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.AppList{})

			helmReleaseGVR := schema.GroupVersionResource{
				Group:    "helm.toolkit.fluxcd.io",
				Version:  "v2beta1",
				Resource: "helmreleases",
			}

			c := Config{
				IndexCache: indexcachetest.New(indexcachetest.Config{
					GetIndexResponse: newIndexWithApp("existing-app", "1.0.0", "https://giantswarm.github.io/app-catalog/existing-app-1.0.0.tgz"),
				}),
				Logger:     microloggertest.New(),
				CtrlClient: fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build(),
				DynamicClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
					runtime.NewScheme(),
					map[schema.GroupVersionResource]string{
						helmReleaseGVR: "HelmReleaseList",
					},
					unstructuredObjs...),

				ChartNamespace:               "giantswarm",
				DependencyWaitTimeoutMinutes: 30,
			}

			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			ctx := context.Background()
			dependenciesNotInstalledResult, err := r.checkDependencies(ctx, *tc.appToInstall)
			if err != nil {
				errorMessage := err.Error()
				expectedErrorMessageStart := fmt.Sprintf("Not creating chart for app %q: dependencies not satisfied", tc.appToInstall.Name)
				if strings.Index(errorMessage, expectedErrorMessageStart) != 0 {
					t.Fatal(err)
				}
			}

			if len(tc.dependenciesNotInstalled) != len(dependenciesNotInstalledResult) {
				t.Fatalf(
					"expected %d not installed dependencies are [%s], got %d [%s]",
					len(tc.dependenciesNotInstalled),
					strings.Join(tc.dependenciesNotInstalled, ","),
					len(dependenciesNotInstalledResult),
					strings.Join(dependenciesNotInstalledResult, ","))
			}

			sort.Strings(tc.dependenciesNotInstalled)
			sort.Strings(dependenciesNotInstalledResult)
			for i, expectedDependencyNotInstalled := range tc.dependenciesNotInstalled {
				if expectedDependencyNotInstalled != dependenciesNotInstalledResult[i] {
					t.Fatalf("expected not installed dependencies are [%s], got [%s]", strings.Join(tc.dependenciesNotInstalled, ","), strings.Join(dependenciesNotInstalledResult, ","))
				}
			}
		})
	}
}
