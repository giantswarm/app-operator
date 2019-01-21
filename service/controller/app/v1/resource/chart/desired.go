package chart

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	catalogName := key.CatalogName(cr)

	appCatalog, err := r.g8sClient.ApplicationV1alpha1().AppCatalogs(r.watchNamespace).Get(catalogName, v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "appCatalog %#q in namespace %#q", catalogName, "default")
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	chartURL, err := generateCatalogURL(appCatalog.Spec.CatalogStorage.URL, cr.Spec.Name, cr.Spec.Release)
	if err != nil {
		return nil, err
	}

	chartCR := &v1alpha1.Chart{
		TypeMeta: v1.TypeMeta{
			Kind:       chartKind,
			APIVersion: chartAPIVersion,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:        cr.Spec.Name,
			Labels:      cr.GetObjectMeta().GetLabels(),
			Annotations: cr.GetObjectMeta().GetAnnotations(),
		},
		Spec: v1alpha1.ChartSpec{
			Name:      cr.GetObjectMeta().GetName(),
			Namespace: cr.Spec.Namespace,
			Config: v1alpha1.ChartSpecConfig{
				ConfigMap: v1alpha1.ChartSpecConfigConfigMap{
					Name:      key.ConfigMapName(cr),
					Namespace: key.ConfigMapNamespace(cr),
				},
				Secret: v1alpha1.ChartSpecConfigSecret{
					Name:      key.SecretName(cr),
					Namespace: key.SecretNamespace(cr),
				},
			},
			TarballURL: chartURL,
		},
	}

	return chartCR, nil
}

func generateCatalogURL(baseURL string, appName string, release string) (string, error) {
	if baseURL == "" || appName == "" || release == "" {
		return "", microerror.Maskf(failedExecution, "baseURL(%s), appName(%s), release(%s) should not left as blank string", baseURL, appName, release)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", microerror.Mask(err)
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("%s-%s.tgz", appName, release))
	return u.String(), nil
}
