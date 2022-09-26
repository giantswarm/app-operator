package chart

import (
	"context"
	"net/url"
	"path"
	"strings"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/v6/pkg/project"
	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache"
)

const (
	chartPullFailedStatus = "chart-pull-failed"
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

	repositoryURL, err := r.pickRepositoryURL(ctx, cc, cr, chartName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	tarballURL, version, err := r.buildTarballURL(ctx, cc, cr, repositoryURL)
	if err != nil && IsNotFound(err) && key.CatalogVisibility(cc.Catalog) != "internal" {
		// Could not reach custom catalog's index or find app entry in it.
		// Let's retry using other repositories.
		r.logger.Errorf(ctx, err, "failed to resolve tarball URL for %#q repository, trying next one", repositoryURL)
		for _, fallbackURL := range fallbackRepositories(cc.Catalog, repositoryURL) {
			tarballURL, version, err = r.buildTarballURL(ctx, cc, cr, fallbackURL)
			if err == nil {
				r.logger.Debugf(ctx, "found a working tarball URL in repository %#q", fallbackURL)
				break
			} else {
				r.logger.Errorf(ctx, err, "failed to resolve tarball URL for %#q repository, trying next one", fallbackURL)
			}
		}
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else if err != nil {
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
			Rollback:   generateRollback(cr),
			Uninstall:  generateUninstall(cr),
			Upgrade:    generateUpgrade(cr),
			TarballURL: tarballURL,
			Version:    version,
		},
	}

	return chartCR, nil
}

func (r *Resource) pickRepositoryURL(ctx context.Context, cc *controllercontext.Context, cr v1alpha1.App, chartName string) (string, error) {
	switch len(cc.Catalog.Spec.Repositories) {
	case 0:
		return cc.Catalog.Spec.Storage.URL, nil
	case 1:
		return cc.Catalog.Spec.Repositories[0].URL, nil
	}

	var chart v1alpha1.Chart
	err := cc.Clients.K8s.CtrlClient().Get(
		ctx,
		types.NamespacedName{Name: chartName, Namespace: r.chartNamespace},
		&chart,
	)
	if apierrors.IsNotFound(err) || tenant.IsAPINotAvailable(err) {
		// Repositories is guaranteed by Custom Resource Definition to have at least one entry.
		return cc.Catalog.Spec.Repositories[0].URL, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	// Check currently selected repository
	repositoryIndex := -1
	for i, repo := range cc.Catalog.Spec.Repositories {
		if strings.Contains(chart.Spec.TarballURL, repo.URL) {
			repositoryIndex = i
			break
		}
	}
	if repositoryIndex == -1 {
		// Could not match current tarballURL to any of Catalog's repositories.
		// Maybe the list was updated. Let's pick any existing repository.
		r.logger.Debugf(ctx, "could not match tarball URL %q to any of %q Catalog repositories; using default", chart.Spec.TarballURL, cc.Catalog.Name)
		return cc.Catalog.Spec.Repositories[0].URL, nil
	}

	if chart.Status.Release.Status == chartPullFailedStatus {
		// chart-operator had trouble pulling the chart -- this includes timeouts and chart not being found (404)
		// Round-robin the repository.
		repositoryIndex = (repositoryIndex + 1) % len(cc.Catalog.Spec.Repositories)
	}
	return cc.Catalog.Spec.Repositories[repositoryIndex].URL, nil
}

func (r *Resource) buildTarballURL(ctx context.Context, cc *controllercontext.Context, cr v1alpha1.App, repositoryURL string) (url string, version string, err error) {
	if key.CatalogVisibility(cc.Catalog) == "internal" {
		// For internal catalogs we generate the URL as its predictable
		// and to avoid having chicken egg problems.
		url, err = appcatalog.NewTarballURL(repositoryURL, key.AppName(cr), key.Version(cr))
		if err != nil {
			return "", "", microerror.Mask(err)
		}
		version = key.Version(cr)
		return url, version, nil
	}

	// For all other catalogs we check the index.yaml for compatibility
	// with community catalogs.
	index, err := r.indexCache.GetIndex(ctx, repositoryURL)
	if err != nil {
		r.logger.Errorf(ctx, err, "failed to get index.yaml from %q", repositoryURL)
	}
	if index == nil {
		return "", "", microerror.Maskf(notFoundError, "index %#v for %q is <nil>", index, repositoryURL)
	}
	if len(index.Entries) == 0 {
		return "", "", microerror.Maskf(notFoundError, "index %#v for %q has no entries", index, repositoryURL)
	}

	entries, ok := index.Entries[cr.Spec.Name]
	if !ok {
		return "", "", microerror.Maskf(notFoundError, "no entries for app %#q in index.yaml for %q", cr.Spec.Name, repositoryURL)
	}

	// We first try with the full version set in .spec.version of the app CR.
	version = cr.Spec.Version
	url, err = getEntryURL(entries, cr.Spec.Name, version)
	if err != nil {
		// We try again without the `v` prefix. This enables us to use the
		// Flux Image Automation controller to automatically update apps.
		version = strings.TrimPrefix(version, "v")

		url, err = getEntryURL(entries, cr.Spec.Name, version)
		if err != nil {
			return "", "", microerror.Mask(err)
		}
	}

	if url == "" {
		return "", "", microerror.Maskf(notFoundError, "found entry for app %#q but URL is not specified", cr.Spec.Name)
	}

	if !isValidURL(url) {
		// URL may be relative. If so we join it to the Catalog Storage URL.
		url, err = joinRelativeURL(repositoryURL, url)
		if err != nil {
			return "", "", microerror.Mask(err)
		}
	}

	return url, version, err
}

func fallbackRepositories(catalog v1alpha1.Catalog, repositoryURL string) []string {
	urls := []string{}
	repositoryIndex := -1
	for i, repo := range catalog.Spec.Repositories {
		if repo.URL == repositoryURL {
			repositoryIndex = i
		}
		urls = append(urls, repo.URL)
	}
	if repositoryIndex == -1 {
		// could not find failed repositoryURL, let's just return the whole slice
		return urls
	}

	// Return all repositoryURLs, starting with the one after repositoryURL and skip repositoryURL.
	// example: urls=["a", "b", "c", "d"], repositoryURL="c" -> ["d", "a", "b"]
	// example: urls=["x"], repositoryURL="x" -> []
	return append(urls[repositoryIndex+1:], urls[:repositoryIndex]...)
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
	install := v1alpha1.ChartSpecInstall{}

	if key.InstallSkipCRDs(cr) {
		install.SkipCRDs = true
	}

	timeout := key.InstallTimeout(cr)
	if timeout != nil {
		install.Timeout = timeout
	}

	return install
}

func generateRollback(cr v1alpha1.App) v1alpha1.ChartSpecRollback {
	rollback := v1alpha1.ChartSpecRollback{}

	timeout := key.RollbackTimeout(cr)
	if timeout != nil {
		rollback.Timeout = timeout
	}

	return rollback
}

func generateUninstall(cr v1alpha1.App) v1alpha1.ChartSpecUninstall {
	uninstall := v1alpha1.ChartSpecUninstall{}

	timeout := key.UninstallTimeout(cr)
	if timeout != nil {
		uninstall.Timeout = timeout
	}

	return uninstall
}

func generateUpgrade(cr v1alpha1.App) v1alpha1.ChartSpecUpgrade {
	upgrade := v1alpha1.ChartSpecUpgrade{}

	timeout := key.UpgradeTimeout(cr)
	if timeout != nil {
		upgrade.Timeout = timeout
	}

	return upgrade
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
	if key.AppConfigMapName(cr) != "" || key.CatalogConfigMapName(catalog) != "" || key.UserConfigMapName(cr) != "" || hasKindInExtraConfigs(cr, "configMap") {
		return true
	}

	return false
}

func hasSecret(cr v1alpha1.App, catalog v1alpha1.Catalog) bool {
	if key.AppSecretName(cr) != "" || key.CatalogSecretName(catalog) != "" || key.UserSecretName(cr) != "" || hasKindInExtraConfigs(cr, "secret") {
		return true
	}

	return false
}

func hasKindInExtraConfigs(cr v1alpha1.App, kind string) bool {
	kindLowerCase := strings.ToLower(kind)

	for _, extraConfig := range key.ExtraConfigs(cr) {
		if strings.ToLower(extraConfig.Kind) == kindLowerCase {
			return true
		}
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

func joinRelativeURL(baseURL, relativeURL string) (string, error) {
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
