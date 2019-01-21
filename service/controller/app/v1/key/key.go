package key

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
)

const (
	versionBundleAnnotation = "giantswarm.io/version-bundle"
)

func AppName(customResource v1alpha1.App) string {
	return customResource.Spec.Name
}

func CatalogName(customResource v1alpha1.App) string {
	return customResource.Spec.Catalog
}

func ConfigMapName(customResource v1alpha1.App) string {
	return customResource.Spec.Config.ConfigMap.Name
}

func ConfigMapNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.Config.ConfigMap.Namespace
}

func Namespace(customResource v1alpha1.App) string {
	return customResource.Spec.Namespace
}

func ReleaseName(customResource v1alpha1.App) string {
	return customResource.Spec.Release
}

func SecretName(customResource v1alpha1.App) string {
	return customResource.Spec.Config.Secret.Name
}

func SecretNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.Config.Secret.Namespace
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
	customResource, ok := v.(*v1alpha1.Chart)
	if !ok {
		return v1alpha1.Chart{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1alpha1.Chart{}, v)
	}

	if customResource == nil {
		return v1alpha1.Chart{}, microerror.Maskf(emptyValueError, "empty value cannot be converted to Chart")
	}

	return *customResource, nil
}

func VersionBundleVersion(customResource v1alpha1.App) string {
	if val, ok := customResource.ObjectMeta.Annotations[versionBundleAnnotation]; ok {
		return val
	} else {
		return ""
	}
}
