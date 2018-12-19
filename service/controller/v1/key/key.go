package key

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
)

const (
	VERSION_BUNDLE = "giantswarm.io/version-bundle"
)

func AppName(customObject v1alpha1.App) string {
	return customObject.Spec.Name
}

func Namespace(customObject v1alpha1.App) string {
	return customObject.Spec.Namespace
}

func ReleaseName(customObject v1alpha1.App) string {
	return customObject.Spec.Release
}

// ToCustomObject converts value to v1alpha1.ChartConfig and returns it or error
// if type does not match.
func ToCustomObject(v interface{}) (v1alpha1.App, error) {
	customObjectPointer, ok := v.(*v1alpha1.App)
	if !ok {
		return v1alpha1.App{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1alpha1.App{}, v)
	}

	if customObjectPointer == nil {
		return v1alpha1.App{}, microerror.Maskf(emptyValueError, "empty value cannot be converted to CustomObject")
	}

	return *customObjectPointer, nil
}

func VersionBundleVersion(customObject v1alpha1.App) string {
	if val, ok := customObject.ObjectMeta.Annotations[VERSION_BUNDLE]; ok {
		return val
	} else {
		return ""
	}
}
