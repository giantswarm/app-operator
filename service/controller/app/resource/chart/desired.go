package chart

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/app/v4/pkg/annotation"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/v2/pkg/project"
	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToApp(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if key.IsDeleted(cr) {
		// Return empty chart CR so it is deleted.
		chartCR := &v1alpha1.Chart{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cr.Name,
				Namespace: r.chartNamespace,
			},
		}

		return chartCR, nil
	}

	config, err := generateConfig(ctx, cc.Clients.K8s.K8sClient(), cr, cc.AppCatalog, r.chartNamespace)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	tarballURL, err := appcatalog.NewTarballURL(key.AppCatalogStorageURL(cc.AppCatalog), key.AppName(cr), key.Version(cr))
	if err != nil {
		r.logger.Errorf(ctx, err, "failed to generated tarball")
	}

	chartCR := &v1alpha1.Chart{
		TypeMeta: metav1.TypeMeta{
			Kind:       chartKind,
			APIVersion: chartAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.GetName(),
			Namespace: r.chartNamespace,
			Labels:    processLabels(project.Name(), cr.GetLabels()),
		},
		Spec: v1alpha1.ChartSpec{
			Config:     config,
			Name:       cr.GetName(),
			Namespace:  key.Namespace(cr),
			TarballURL: tarballURL,
			Version:    key.Version(cr),
		},
	}

	annotations := generateAnnotations(cr.GetAnnotations())

	u, err := url.Parse(r.webhookBaseURL)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	u.Path = path.Join(u.Path, "status", cr.Namespace, cr.Name)

	webhookAnnotation := fmt.Sprintf("%s/%s", annotation.ChartOperatorPrefix, annotation.WebhookURL)
	annotations[webhookAnnotation] = u.String()

	if len(annotations) > 0 {
		chartCR.Annotations = annotations
	}

	return chartCR, nil
}

func generateAnnotations(input map[string]string) map[string]string {
	annotations := map[string]string{}

	for k, v := range input {
		// Copy all annotations which has a prefix with chart-operator.giantswarm.io.
		if strings.HasPrefix(k, annotation.ChartOperatorPrefix) {
			annotations[k] = v
		}
	}

	return annotations
}

func generateConfig(ctx context.Context, k8sClient kubernetes.Interface, cr v1alpha1.App, appCatalog v1alpha1.AppCatalog, chartNamespace string) (v1alpha1.ChartSpecConfig, error) {
	config := v1alpha1.ChartSpecConfig{}

	if hasConfigMap(cr, appCatalog) {
		configMapName := key.ChartConfigMapName(cr)
		cm, err := k8sClient.CoreV1().ConfigMaps(chartNamespace).Get(ctx, configMapName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// no-op
		} else if err != nil {
			return v1alpha1.ChartSpecConfig{}, microerror.Mask(err)
		} else {
			configMap := v1alpha1.ChartSpecConfigConfigMap{
				Name:            configMapName,
				Namespace:       chartNamespace,
				ResourceVersion: cm.GetResourceVersion(),
			}

			config.ConfigMap = configMap
		}
	}

	if hasSecret(cr, appCatalog) {
		secretName := key.ChartSecretName(cr)
		secret, err := k8sClient.CoreV1().Secrets(chartNamespace).Get(ctx, secretName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// no-op
		} else if err != nil {
			return v1alpha1.ChartSpecConfig{}, microerror.Mask(err)
		} else {
			secretConfig := v1alpha1.ChartSpecConfigSecret{
				Name:            secretName,
				Namespace:       chartNamespace,
				ResourceVersion: secret.GetResourceVersion(),
			}

			config.Secret = secretConfig
		}
	}

	return config, nil
}

func hasConfigMap(cr v1alpha1.App, appCatalog v1alpha1.AppCatalog) bool {
	if key.AppConfigMapName(cr) != "" || key.AppCatalogConfigMapName(appCatalog) != "" || key.UserConfigMapName(cr) != "" {
		return true
	}

	return false
}

func hasSecret(cr v1alpha1.App, appCatalog v1alpha1.AppCatalog) bool {
	if key.AppSecretName(cr) != "" || key.AppCatalogSecretName(appCatalog) != "" || key.UserSecretName(cr) != "" {
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
