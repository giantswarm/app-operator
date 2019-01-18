package chart

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customResource, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	catalogName := key.CatalogName(customResource)

	appCatalog, err := r.g8sClient.ApplicationV1alpha1().AppCatalogs("default").Get(catalogName, v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "appCatalog '%s' in namespace 'default' not found", catalogName)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	chartURL := generateCatalogURL(appCatalog.Spec.CatalogStorage.URL, customResource.Spec.Name, customResource.Spec.Release)

	chartCR := &v1alpha1.Chart{
		TypeMeta: v1.TypeMeta{
			Kind:       chartKind,
			APIVersion: chartAPIVersion,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:        customResource.Spec.Name,
			Labels:      customResource.GetObjectMeta().GetLabels(),
			Annotations: customResource.GetObjectMeta().GetAnnotations(),
		},
		Spec: v1alpha1.ChartSpec{
			Name:       customResource.GetObjectMeta().GetName(),
			Namespace:  customResource.Spec.Namespace,
			TarballURL: chartURL,
		},
	}

	if customResource.Spec.KubeConfig != (v1alpha1.AppSpecKubeConfig{}) {
		chartCR.Spec.KubeConfig.Secret.Name = customResource.Spec.KubeConfig.Secret.Name
		chartCR.Spec.KubeConfig.Secret.Namespace = customResource.Spec.KubeConfig.Secret.Namespace
	}

	if customResource.Spec.Config != (v1alpha1.AppSpecConfig{}) {
		chartCR.Spec.Config.Secret.Name = customResource.Spec.Config.Secret.Name
		chartCR.Spec.Config.Secret.Namespace = customResource.Spec.Config.Secret.Namespace

		chartCR.Spec.Config.ConfigMap.Name = customResource.Spec.Config.ConfigMap.Name
		chartCR.Spec.Config.ConfigMap.Namespace = customResource.Spec.Config.ConfigMap.Namespace
	}
	return chartCR, nil
}

func generateCatalogURL(baseURL string, appName string, release string) string {
	return fmt.Sprintf("%s-%s-%s.tgz", baseURL, appName, release)
}
