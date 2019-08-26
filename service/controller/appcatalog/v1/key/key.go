package key

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/pkg/label"
)

func AppCatalogTitle(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Title
}

func AppCatalogStorageURL(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Storage.URL
}

func ConfigMapName(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Config.ConfigMap.Name
}

func ConfigMapNamespace(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Config.ConfigMap.Namespace
}

func IsDeleted(cr v1alpha1.AppCatalog) bool {
	return cr.DeletionTimestamp != nil
}

func SecretName(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Config.Secret.Name
}

func SecretNamespace(customResource v1alpha1.AppCatalog) string {
	return customResource.Spec.Config.Secret.Namespace
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

// VersionLabel returns the label value to determine if the custom resource is
// supported by this version of the operatorkit resource.
func VersionLabel(customResource v1alpha1.AppCatalog) string {
	if val, ok := customResource.ObjectMeta.Labels[label.AppOperatorVersion]; ok {
		return val
	} else {
		return ""
	}
}
