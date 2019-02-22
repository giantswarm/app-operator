package key

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
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

// ToConfigMap converts value to corev1.ConfigMap and returns it or error
// if type does not match.
func ToConfigMap(v interface{}) (corev1.ConfigMap, error) {
	customResource, ok := v.(*corev1.ConfigMap)
	if !ok {
		return corev1.ConfigMap{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.ConfigMap{}, v)
	}

	if customResource == nil {
		return corev1.ConfigMap{}, microerror.Maskf(emptyValueError, "empty value cannot be converted to CustomObject")
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
