package chart

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/annotation"
	"github.com/giantswarm/app-operator/pkg/label"
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

	config := generateConfig(cr, cc.AppCatalog, r.chartNamespace)
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
			Annotations: processAnnotations(cr),
			Name:        cr.GetName(),
			Namespace:   r.chartNamespace,
			Labels:      processLabels(r.projectName, cr.GetLabels()),
		},
		Spec: v1alpha1.ChartSpec{
			Config:     config,
			Name:       cr.GetName(),
			Namespace:  key.Namespace(cr),
			TarballURL: tarballURL,
		},
	}

	return chartCR, nil
}

func generateConfig(cr v1alpha1.App, appCatalog v1alpha1.AppCatalog, chartNamespace string) v1alpha1.ChartSpecConfig {
	config := v1alpha1.ChartSpecConfig{}

	if hasConfigMap(cr, appCatalog) {
		configMap := v1alpha1.ChartSpecConfigConfigMap{
			Name:      key.ChartConfigMapName(cr),
			Namespace: chartNamespace,
		}

		config.ConfigMap = configMap
	}

	if hasSecret(cr, appCatalog) {
		secret := v1alpha1.ChartSpecConfigSecret{
			Name:      key.ChartSecretName(cr),
			Namespace: chartNamespace,
		}

		config.Secret = secret
	}

	return config
}

func generateTarballURL(baseURL string, appName string, version string) (string, error) {
	if baseURL == "" || appName == "" || version == "" {
		return "", microerror.Maskf(executionFailedError, "baseURL %#q, appName %#q, release %#q should not be empty", baseURL, appName, version)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", microerror.Mask(err)
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("%s-%s.tgz", appName, version))
	return u.String(), nil
}

func hasConfigMap(cr v1alpha1.App, appCatalog v1alpha1.AppCatalog) bool {
	if key.AppConfigMapName(cr) != "" || appcatalogkey.ConfigMapName(appCatalog) != "" {
		return true
	}

	return false
}

func hasSecret(cr v1alpha1.App, appCatalog v1alpha1.AppCatalog) bool {
	if key.AppSecretName(cr) != "" || appcatalogkey.SecretName(appCatalog) != "" {
		return true
	}

	return false
}

func replacePrefix(from string) string {
	return strings.ReplaceAll(from, "app-operator.giantswarm.io", "chart-operator.giantswarm.io")
}

// processAnnotations ensures necessary app-operator.giantswarm.io annotations
// are copied into the proper prefix in chart CR.
func processAnnotations(cr v1alpha1.App) map[string]string {

	if cr.GetAnnotations() == nil {
		return nil
	}

	newAnnotations := map[string]string{}

	if key.IsCordoned(cr) {
		cordonReasonKey := replacePrefix(annotation.CordonReason)
		newAnnotations[cordonReasonKey] = key.CordonReason(cr)

		cordonUntilKey := replacePrefix(annotation.CordonUntilDate)
		newAnnotations[cordonUntilKey] = key.CordonUntil(cr)
	}

	return newAnnotations
}

// processLabels ensures the chart-operator.giantswarm.io/version label is
// present and the app-operator.giantswarm.io/version label is removed. It
// also ensures the giantswarm.io/managed-by label is accurate.
//
// Any other labels added to the app custom resource are passed on to the chart
// custom resource.
func processLabels(projectName string, inputLabels map[string]string) map[string]string {
	// These labels are required.
	labels := map[string]string{
		label.ChartOperatorVersion: chartCustomResourceVersion,
		label.ManagedBy:            projectName,
	}

	for k, v := range inputLabels {
		// These labels must be removed.
		if k != label.ManagedBy && k != label.AppOperatorVersion {
			labels[k] = v
		}
	}

	return labels
}
