package chart

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/pkg/tarball"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
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

	if cc.AppCatalog.Name == "" {
		return nil, nil
	}

	config := generateConfig(cr, cc.AppCatalog, r.chartNamespace)
	tarballURL, err := tarball.NewURL(key.AppCatalogStorageURL(cc.AppCatalog), key.AppName(cr), key.Version(cr))
	if err != nil {
		return nil, err
	}

	chartCR := &v1alpha1.Chart{
		TypeMeta: metav1.TypeMeta{
			Kind:       chartKind,
			APIVersion: chartAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.GetName(),
			Namespace: r.chartNamespace,
			Labels:    processLabels(r.projectName, cr.GetLabels()),
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

func hasConfigMap(cr v1alpha1.App, appCatalog v1alpha1.AppCatalog) bool {
	if key.AppConfigMapName(cr) != "" || key.AppCatalogConfigMapName(appCatalog) != "" {
		return true
	}

	return false
}

func hasSecret(cr v1alpha1.App, appCatalog v1alpha1.AppCatalog) bool {
	if key.AppSecretName(cr) != "" || key.AppCatalogSecretName(appCatalog) != "" {
		return true
	}

	return false
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
