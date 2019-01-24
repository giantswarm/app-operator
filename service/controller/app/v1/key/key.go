package key

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/pkg/label"
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

func KubeConfigSecretName(customResource v1alpha1.App) string {
	return customResource.Spec.KubeConfig.Secret.Name
}

func KubeConfigSecretNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.KubeConfig.Secret.Namespace
}

func Namespace(customResource v1alpha1.App) string {
	return customResource.Spec.Namespace
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
