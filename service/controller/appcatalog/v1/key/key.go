package key

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
)

const (
	versionBundleAnnotation = "giantswarm.io/version-bundle"
)

func AppCatalogTitle(customObject v1alpha1.AppCatalog) string {
	return customObject.Spec.Title
}

// ToCustomResource converts value to v1alpha1.AppCatalog and returns it or error
// if type does not match.
func ToCustomResource(v interface{}) (v1alpha1.AppCatalog, error) {
	customResource, ok := v.(*v1alpha1.AppCatalog)
	if !ok {
		return v1alpha1.AppCatalog{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1alpha1.AppCatalog{}, v)
	}

	if customResource == nil {
		return v1alpha1.AppCatalog{}, microerror.Maskf(emptyValueError, "empty value cannot be converted to CustomObject")
	}

	return *customResource, nil
}

func VersionBundleVersion(customObject v1alpha1.AppCatalog) string {
	if val, ok := customObject.ObjectMeta.Annotations[versionBundleAnnotation]; ok {
		return val
	} else {
		return ""
	}
}
