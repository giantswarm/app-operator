package key

import (
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/app-operator/v2/pkg/annotation"
	"github.com/giantswarm/app-operator/v2/pkg/project"
)

const (
	ChartOperatorAppName = "chart-operator"
)

func AppCatalogTitle(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Title
}

func AppCatalogStorageURL(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Storage.URL
}

func AppCatalogConfigMapName(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Config.ConfigMap.Name
}

func AppCatalogConfigMapNamespace(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Config.ConfigMap.Namespace
}

func AppCatalogSecretName(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Config.Secret.Name
}

func AppCatalogSecretNamespace(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Config.Secret.Namespace
}

func AppConfigMapName(customResource v1alpha1.App) string {
	return customResource.Spec.Config.ConfigMap.Name
}

func AppConfigMapNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.Config.ConfigMap.Namespace
}

func AppName(customResource v1alpha1.App) string {
	return customResource.Spec.Name
}

func AppSecretName(customResource v1alpha1.App) string {
	return customResource.Spec.Config.Secret.Name
}

func AppSecretNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.Config.Secret.Namespace
}

func AppStatus(customResource v1alpha1.App) v1alpha1.AppStatus {
	return customResource.Status
}

func AppVersionSelector(unique bool) labels.Selector {
	var version string

	version = project.Version()
	if unique {
		// When app-operator is deployed as a unique app it only processes
		// control plane app CRs. These CRs always have the version label
		// app-operator.giantswarm.io/version: 0.0.0
		version = project.AppControlPlaneVersion()
	}
	s := fmt.Sprintf("%s=%s", label.AppOperatorVersion, version)

	selector, err := labels.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse selector %#q with error %#q", s, err))
	}

	return selector
}

func CatalogName(customResource v1alpha1.App) string {
	return customResource.Spec.Catalog
}

func ChartStatus(customResource v1alpha1.Chart) v1alpha1.ChartStatus {
	return customResource.Status
}

func ChartConfigMapName(customResource v1alpha1.App) string {
	return fmt.Sprintf("%s-chart-values", customResource.GetName())
}

func ChartSecretName(customResource v1alpha1.App) string {
	return fmt.Sprintf("%s-chart-secrets", customResource.GetName())
}

func ClusterID(customResource v1alpha1.App) string {
	return customResource.GetLabels()[label.Cluster]
}

func ClusterValuesConfigMapName(customResource v1alpha1.App) string {
	return fmt.Sprintf("%s-cluster-values", customResource.GetNamespace())
}

func CordonReason(customResource v1alpha1.App) string {
	return customResource.GetAnnotations()[fmt.Sprintf("%s/%s", annotation.ChartOperatorPrefix, annotation.CordonReason)]
}

func CordonUntil(customResource v1alpha1.App) string {
	return customResource.GetAnnotations()[fmt.Sprintf("%s/%s", annotation.ChartOperatorPrefix, annotation.CordonUntil)]
}

func CordonUntilDate() string {
	return time.Now().Add(1 * time.Hour).Format("2006-01-02T15:04:05")
}

func DefaultCatalogStorageURL() string {
	return "https://giantswarm.github.com/default-catalog"
}

func InCluster(customResource v1alpha1.App) bool {
	return customResource.Spec.KubeConfig.InCluster
}

func IsAppCordoned(customResource v1alpha1.App) bool {
	_, reasonOk := customResource.Annotations[fmt.Sprintf("%s/%s", annotation.AppOperatorPrefix, annotation.CordonReason)]
	_, untilOk := customResource.Annotations[fmt.Sprintf("%s/%s", annotation.AppOperatorPrefix, annotation.CordonUntil)]

	if reasonOk && untilOk {
		return true
	} else {
		return false
	}
}

func IsChartCordoned(customResource v1alpha1.Chart) bool {
	_, reasonOk := customResource.Annotations[fmt.Sprintf("%s/%s", annotation.ChartOperatorPrefix, annotation.CordonReason)]
	_, untilOk := customResource.Annotations[fmt.Sprintf("%s/%s", annotation.ChartOperatorPrefix, annotation.CordonUntil)]

	if reasonOk && untilOk {
		return true
	} else {
		return false
	}
}

func IsDeleted(customResource v1alpha1.App) bool {
	return customResource.DeletionTimestamp != nil
}

func KubeConfigFinalizer(customResource v1alpha1.App) string {
	return fmt.Sprintf("app-operator.giantswarm.io/app-%s", customResource.GetName())
}

func KubecConfigSecretName(customResource v1alpha1.App) string {
	return customResource.Spec.KubeConfig.Secret.Name
}

func KubecConfigSecretNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.KubeConfig.Secret.Namespace
}

func Namespace(customResource v1alpha1.App) string {
	return customResource.Spec.Namespace
}

func OrganizationID(customResource v1alpha1.App) string {
	return customResource.GetLabels()[label.Organization]
}

func ReleaseName(customResource v1alpha1.App) string {
	return customResource.Spec.Name
}

func ToApp(v interface{}) (v1alpha1.App, error) {
	customResource, ok := v.(*v1alpha1.App)
	if !ok {
		return v1alpha1.App{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1alpha1.App{}, v)
	}

	if customResource == nil {
		return v1alpha1.App{}, microerror.Maskf(emptyValueError, "empty value cannot be converted to customResource")
	}

	return *customResource, nil
}

func UserConfigMapName(customResource v1alpha1.App) string {
	return customResource.Spec.UserConfig.ConfigMap.Name
}

func UserConfigMapNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.UserConfig.ConfigMap.Namespace
}

func UserSecretName(customResource v1alpha1.App) string {
	return customResource.Spec.UserConfig.Secret.Name
}

func UserSecretNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.UserConfig.Secret.Namespace
}

func Version(customResource v1alpha1.App) string {
	return customResource.Spec.Version
}

func VersionLabel(customResource v1alpha1.App) string {
	if val, ok := customResource.ObjectMeta.Labels[label.AppOperatorVersion]; ok {
		return val
	} else {
		return ""
	}
}
