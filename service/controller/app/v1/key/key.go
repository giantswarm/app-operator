package key

import (
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/pkg/label"
)

// AppConfigMapName returns the name of the configmap that stores app level
// config for the provided app CR.
func AppConfigMapName(customResource v1alpha1.App) string {
	return customResource.Spec.Config.ConfigMap.Name
}

// AppConfigMapNamespace returns the namespace of the configmap that stores app
// level config for the provided app CR.
func AppConfigMapNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.Config.ConfigMap.Namespace
}

func AppName(customResource v1alpha1.App) string {
	return customResource.Spec.Name
}

// AppSecretName returns the name of the secret that stores app level
// secrets for the provided app CR.
func AppSecretName(customResource v1alpha1.App) string {
	return customResource.Spec.Config.Secret.Name
}

// AppSecretNamespace returns the namespace of the secret that stores app
// level secrets for the provided app CR.
func AppSecretNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.Config.Secret.Namespace
}

func AppStatus(customResource v1alpha1.App) v1alpha1.AppStatus {
	return customResource.Status
}

func CatalogName(customResource v1alpha1.App) string {
	return customResource.Spec.Catalog
}

func ChartStatus(customResource v1alpha1.Chart) v1alpha1.ChartStatus {
	return customResource.Status
}

// ChartConfigMapName returns the name of the configmap that stores config for
// the chart CR that is generated for the provided app CR.
func ChartConfigMapName(customResource v1alpha1.App) string {
	return fmt.Sprintf("%s-chart-values", customResource.GetName())
}

// ChartSecretName returns the name of the secret that stores secrets for
// the chart CR that is generated for the provided app CR.
func ChartSecretName(customResource v1alpha1.App) string {
	return fmt.Sprintf("%s-chart-secrets", customResource.GetName())
}

func InCluster(customResource v1alpha1.App) bool {
	return customResource.Spec.KubeConfig.InCluster
}

func Namespace(customResource v1alpha1.App) string {
	return customResource.Spec.Namespace
}

// ToCustomResource converts value to v1alpha1.App and returns it or error
// if type does not match.
func ToCustomResource(v interface{}) (v1alpha1.App, error) {
	customResource, ok := v.(*v1alpha1.App)
	if !ok {
		return v1alpha1.App{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1alpha1.App{}, v)
	}

	if customResource == nil {
		return v1alpha1.App{}, microerror.Maskf(emptyValueError, "empty value cannot be converted to customResource")
	}

	return *customResource, nil
}

func ToChart(v interface{}) (v1alpha1.Chart, error) {
	if v == nil {
		return v1alpha1.Chart{}, nil
	}
	customResource, ok := v.(*v1alpha1.Chart)
	if !ok {
		return v1alpha1.Chart{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1alpha1.Chart{}, v)
	}

	return *customResource, nil
}

func Version(customResource v1alpha1.App) string {
	return customResource.Spec.Version
}

// VersionLabel returns the label value to determine if the custom resource is
// supported by this version of the operatorkit resource.
func VersionLabel(customResource v1alpha1.App) string {
	if val, ok := customResource.ObjectMeta.Labels[label.AppOperatorVersion]; ok {
		return val
	} else {
		return ""
	}
}
