package chart

import (
	"context"
	"fmt"
	"github.com/giantswarm/app-operator/pkg/label"
	"net/url"
	"path"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	appcatalogkey "github.com/giantswarm/app-operator/service/controller/appcatalog/v1/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	tarballURL, err := generateTarballURL(appcatalogkey.AppCatalogStorageURL(cc.AppCatalog), key.AppName(cr), key.Version(cr))
	if err != nil {
		return nil, err
	}

	chartCR := &v1alpha1.Chart{
		TypeMeta: metav1.TypeMeta{
			Kind:       chartKind,
			APIVersion: chartAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cr.Spec.Name,
			Labels: label.ProcessLabels(cr.ObjectMeta.Labels,
				map[string]string{
					label.ManagedBy:            r.projectName,
					label.ChartOperatorVersion: chartCustomResourceVersion,
				},
				map[string]string{label.AppOperatorVersion: ""}),
			Annotations: cr.ObjectMeta.Annotations,
		},
		Spec: v1alpha1.ChartSpec{
			Name:      cr.ObjectMeta.Name,
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
			TarballURL: tarballURL,
		},
	}

	return chartCR, nil
}

func generateTarballURL(baseURL string, appName string, version string) (string, error) {
	if baseURL == "" || appName == "" || version == "" {
		return "", microerror.Maskf(failedExecution, "baseURL %#q, appName %#q, release %#q should not be empty", baseURL, appName, version)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", microerror.Mask(err)
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("%s-%s.tgz", appName, version))
	return u.String(), nil
}
