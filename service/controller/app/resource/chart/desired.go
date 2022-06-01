package chart

import (
	"context"
	"net/url"
	"path"
	"strings"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/v5/pkg/project"
	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v5/service/internal/indexcache"
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

	chartName := key.ChartName(cr, r.workloadClusterID)

	config, err := generateConfig(ctx, cc.Clients.K8s.K8sClient(), cr, cc.Catalog, r.chartNamespace)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	tarballURL, version, err := r.buildTarballURL(ctx, cc, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	chartCR := &v1alpha1.Chart{
		TypeMeta: metav1.TypeMeta{
			Kind:       chartKind,
			APIVersion: chartAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        chartName,
			Namespace:   r.chartNamespace,
			Annotations: generateAnnotations(cr.GetAnnotations(), cr.Namespace, cr.Name),
			Labels:      processLabels(project.Name(), cr.GetLabels()),
		},
		Spec: v1alpha1.ChartSpec{
			Config:    config,
			Install:   generateInstall(cr),
			Name:      chartName,
			Namespace: key.Namespace(cr),
			NamespaceConfig: v1alpha1.ChartSpecNamespaceConfig{
				Annotations: cr.Spec.NamespaceConfig.Annotations,
				Labels:      cr.Spec.NamespaceConfig.Labels,
			},
			TarballURL: tarballURL,
			Version:    version,
		},
	}

	return chartCR, nil
}

func (r *Resource) buildTarballURL(ctx context.Context, cc *controllercontext.Context, cr v1alpha1.App) (url string, version string, err error) {
	if key.CatalogVisibility(cc.Catalog) == "internal" {
		// For internal catalogs we generate the URL as its predictable
		// and to avoid having chicken egg problems.

		// TODO(kuba): key.CatalogStorageURL is gone. Also, we want to rotate
		// over .spec.repositories and check (HEAD request) if the tarball
		// exists and is reachable there.
		// TODO(kuba): How to check ORAS registry?

		url, err = appcatalog.NewTarballURL(key.CatalogStorageURL(cc.Catalog), key.AppName(cr), key.Version(cr))
		if err != nil {
			return "", "", microerror.Mask(err)
		}
		version = key.Version(cr)
		return url, version, nil
	}

	// For all other catalogs we check the index.yaml for compatibility
	// with community catalogs.

	// TODO(kuba): Same for index.yaml - check if it's available for download, iterating over storage
	index, err := r.indexCache.GetIndex(ctx, key.CatalogStorageURL(cc.Catalog))
	if err != nil {
		r.logger.Errorf(ctx, err, "failed to get index.yaml")
	}

	if index == nil || len(index.Entries) == 0 {
		return "", "", microerror.Maskf(notFoundError, "no entries in index %#v", index)
	}

	entries, ok := index.Entries[cr.Spec.Name]
	if !ok {
		return "", "", microerror.Maskf(notFoundError, "no entries for app %#q in index.yaml", cr.Spec.Name)
	}

	// We first try with the full version set in .spec.version of the app CR.
	url, err = getEntryURL(entries, cr.Spec.Name, version)
	if err != nil {
		// We try again without the `v` prefix. This enables us to use the
		// Flux Image Automation controller to automatically update apps.
		// TODO(kuba): can we build a list of URLs to try?
		version = strings.TrimPrefix(version, "v")

		url, err = getEntryURL(entries, cr.Spec.Name, version)
		if err != nil {
			return "", "", microerror.Mask(err)
		}
	}

	if !isValidURL(url) {
		// URL may be relative. If so we join it to the Catalog Storage URL.
		// TODO (kuba) this is what produces stupid cut-off urls. Fix it in
		// the process.
		url, err = joinRelativeURL(cc.Catalog, url)
		if err != nil {
			return "", "", microerror.Mask(err)
		}
	}

	return url, version, err
}

func generateAnnotations(input map[string]string, appNamespace, appName string) map[string]string {
	annotations := map[string]string{
		annotation.AppNamespace: appNamespace,
		annotation.AppName:      appName,
	}

	for k, v := range input {
		// Copy all annotations which has a prefix with chart-operator.giantswarm.io.
		if strings.HasPrefix(k, annotation.ChartOperatorPrefix) {
			annotations[k] = v
		}
	}

	return annotations
}

func generateConfig(ctx context.Context, k8sClient kubernetes.Interface, cr v1alpha1.App, catalog v1alpha1.Catalog, chartNamespace string) (v1alpha1.ChartSpecConfig, error) {
	config := v1alpha1.ChartSpecConfig{}

	if hasConfigMap(cr, catalog) {
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

	if hasSecret(cr, catalog) {
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

func generateInstall(cr v1alpha1.App) v1alpha1.ChartSpecInstall {
	if key.InstallSkipCRDs(cr) {
		return v1alpha1.ChartSpecInstall{
			SkipCRDs: true,
		}
	}

	return v1alpha1.ChartSpecInstall{}
}

func getEntryURL(entries []indexcache.Entry, app, version string) (string, error) {
	for _, e := range entries {
		if e.Version == version {
			if len(e.Urls) == 0 {
				return "", microerror.Maskf(notFoundError, "no URL in index.yaml for app %#q version %#q", app, version)
			}

			return e.Urls[0], nil
		}
	}

	return "", microerror.Maskf(notFoundError, "no app %#q in index.yaml with given version %#q", app, version)
}

func hasConfigMap(cr v1alpha1.App, catalog v1alpha1.Catalog) bool {
	if key.AppConfigMapName(cr) != "" || key.CatalogConfigMapName(catalog) != "" || key.UserConfigMapName(cr) != "" {
		return true
	}

	return false
}

func hasSecret(cr v1alpha1.App, catalog v1alpha1.Catalog) bool {
	if key.AppSecretName(cr) != "" || key.CatalogSecretName(catalog) != "" || key.UserSecretName(cr) != "" {
		return true
	}

	return false
}

func isValidURL(input string) bool {
	_, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}

	u, err := url.Parse(input)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

func joinRelativeURL(catalog v1alpha1.Catalog, relativeURL string) (string, error) {
	baseURL := key.CatalogStorageURL(catalog)
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", microerror.Mask(err)
	}

	u.Path = path.Join(u.Path, relativeURL)
	return u.String(), nil
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
