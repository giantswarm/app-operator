package key

import (
	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/v2/pkg/label"
)

func CatalogType(customResource v1alpha1.AppCatalog) string {
	if val, ok := customResource.ObjectMeta.Labels[label.CatalogType]; ok {
		return val
	}

	return ""
}

func CatalogVisibility(customResource v1alpha1.AppCatalog) string {
	if val, ok := customResource.ObjectMeta.Labels[label.CatalogVisibility]; ok {
		return val
	}

	return ""
}

func ToAppCatalog(v interface{}) (v1alpha1.AppCatalog, error) {
	customResource, ok := v.(*v1alpha1.AppCatalog)
	if !ok {
		return v1alpha1.AppCatalog{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1alpha1.AppCatalog{}, v)
	}

	if customResource == nil {
		return v1alpha1.AppCatalog{}, microerror.Maskf(emptyValueError, "empty value cannot be converted to CustomObject")
	}

	return *customResource, nil
}
