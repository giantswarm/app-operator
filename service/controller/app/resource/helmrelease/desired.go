package helmrelease

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appopkey "github.com/giantswarm/app-operator/v6/pkg/key"
	"github.com/giantswarm/app-operator/v6/pkg/project"
	"github.com/giantswarm/app-operator/v6/pkg/status"
	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache"
)

const (
	annotationHelmReleasePauseReason  = "app-operator.giantswarm.io/pause-reason"
	annotationHelmReleasePauseStarted = "app-operator.giantswarm.io/pause-ts"
	annotationAppDependsOn            = "app-operator.giantswarm.io/depends-on"
)

type helmRepository struct {
	repoType string
	repoURL  string
}

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToApp(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// When App CR deletion has been requested, or the
	// cluster is being deleted, return am empty HelmRelease CR,
	// so it gets deleted.
	if key.IsDeleted(cr) || cc.Status.ClusterStatus.IsDeleting {
		helmReleaseCR := &helmv2.HelmRelease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cr.Name,
				Namespace: cr.Namespace,
			},
		}

		return helmReleaseCR, nil
	}

	// helmReleaseCR is preliminary configuration of the HelmRelease CR
	helmReleaseCR := &helmv2.HelmRelease{
		TypeMeta: metav1.TypeMeta{
			Kind:       helmv2.HelmReleaseKind,
			APIVersion: helmv2.GroupVersion.Group,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    processLabels(project.Name(), cr.GetLabels()),
		},
		Spec: helmv2.HelmReleaseSpec{
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:             key.AppName(cr),
					ReconcileStrategy: "ChartVersion",
					Version:           key.Version(cr),
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind:      helmRepositoryKind,
						Namespace: cc.Catalog.Namespace,
					},
				},
			},
			Interval:         metav1.Duration{Duration: 10 * time.Minute},
			Install:          generateInstall(cr),
			Upgrade:          generateUpgrade(cr),
			ReleaseName:      key.ChartName(cr, r.workloadClusterID),
			Rollback:         generateRollback(cr),
			StorageNamespace: key.AppNamespace(cr),
			TargetNamespace:  key.AppNamespace(cr),
			Uninstall:        generateUninstall(cr),
		},
	}

	// Generates HelmRelease .spec.valuesFrom list. This list is to have a single
	// ConfigMap and a single Secret on it only, and reflects the Chart CR config
	// used previously.
	config, version, err := generateConfig(ctx, cc.Clients.K8s.K8sClient(), cr, cc.Catalog)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if len(config) > 0 {
		helmReleaseCR.Spec.ValuesFrom = config
	}

	// The logic below is tasked with choosing the right HelmRepository CR. It
	// follows the same rules as for choosing URL for Chart CR, and returns the
	// same set of errors.
	{
		// Get the primary HelmRepository configuration which is to be the first
		// one to check.
		primary, err := r.pickHelmRepository(ctx, cc, cr)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		repositories := []helmRepository{primary}

		// If the catalog is not internal get remaining repositories that are to
		// play the role of fallback repositories.
		if key.CatalogVisibility(cc.Catalog) != "internal" {
			repositories = append(repositories, fallbackRepositories(cc.Catalog, primary)...)
		}

		// Use the ordered list to find the repository that has the given app
		// inside, and turn it into a valid HelmRepository CR name.
		for _, repo := range repositories {
			err = r.buildTarballURL(ctx, cc, cr, repo)
			if err == nil {
				r.logger.Debugf(ctx, "found a working tarball URL in repository %#q", repo.repoURL)
				// We have found a valid repository the app exists inside. Hence we find a name of
				// the corresponding HelmRepository CR and configure it for the HelmRelease.
				name, err := appopkey.GetHelmRepositoryName(cc.Catalog.Name, repo.repoType, repo.repoURL)
				if err != nil {
					return nil, microerror.Mask(err)
				}
				helmReleaseCR.Spec.Chart.Spec.SourceRef.Name = name
				break
			} else {
				r.logger.Errorf(ctx, err, "failed to resolve tarball URL for %#q repository", repo.repoURL)
			}
		}
		if err != nil {
			setStatus(cc, err)
			resourcecanceledcontext.SetCanceled(ctx)
			return nil, nil
		}
	}

	// The switch from Chart CR to HelmRelease CR makes the Chart Operator
	// annotations and labels mostly pointless. The only supported metadata are now the
	// App Operator annotations related to dependencies and configuration version, with
	// the latter one intended to replace the Chart CR config version information.
	annotations := map[string]string{}
	{
		depsNotInstalled, err := r.checkDependencies(ctx, cr)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if len(depsNotInstalled) > 0 {
			annotations[annotationHelmReleasePauseReason] = fmt.Sprintf("Waiting for dependencies to be installed: %s", strings.Join(depsNotInstalled, ", "))
			annotations[annotationHelmReleasePauseStarted] = time.Now().Format(time.RFC3339)
			helmReleaseCR.Spec.Suspend = true
		}

		// Add config revision information, which was offered as part of
		// the Chart CR spec previously. To offer this information from
		// HelmRelease CR as well, it is from now set as annotation, including
		// reconciliation request annotation for HelmRelease CR.
		//
		// The reconciliation request annotation may be surprising one, but
		// it is the only way to trigger reconciliation of a HelmRelease CR
		// when only values.yaml has changed.

		reconcileRequest := ""

		if v, ok := version[annotation.AppOperatorLatestConfigMapVersion]; ok {
			annotations[annotation.AppOperatorLatestConfigMapVersion] = v
			reconcileRequest += v
		}
		if v, ok := version[annotation.AppOperatorLatestSecretVersion]; ok {
			annotations[annotation.AppOperatorLatestSecretVersion] = v
			reconcileRequest += v
		}
		if reconcileRequest != "" {
			annotations[fluxmeta.ReconcileRequestAnnotation] = reconcileRequest
		}
	}

	if len(annotations) > 0 {
		helmReleaseCR.Annotations = annotations
	}

	// Decide to use the HelmRelease .spec.kubeConfig or not.
	// For in-cluster apps the Helm Controller is to rely on its own
	// service account.
	if !key.InCluster(cr) {
		helmReleaseCR.Spec.KubeConfig = &fluxmeta.KubeConfigReference{
			SecretRef: fluxmeta.SecretKeyReference{
				Name: key.KubeConfigSecretName(cr),
			},
		}
	}

	return helmReleaseCR, nil
}

// checkDependencies is to return a slice with this App CR's dependencies.
func (r *Resource) checkDependencies(ctx context.Context, app v1alpha1.App) ([]string, error) {
	deps, err := getDependenciesFromCR(app)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if len(deps) == 0 {
		r.logger.Debugf(ctx, "App %q has no dependencies", app.Name)
		return nil, nil
	}

	// Get a list of installed and up-to-date apps in the same namespace.
	installedApps := map[string]bool{}
	{
		appList := v1alpha1.AppList{}
		err = r.ctrlClient.List(ctx, &appList, client.InNamespace(app.Namespace))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		for _, app := range appList.Items {
			installedApps[app.Name] = app.Status.Release.Status == "deployed" && app.Status.Version == app.Spec.Version
		}
	}

	// Get a list of dependencies that are not installed.
	dependenciesNotInstalled := make([]string, 0)
	{
		for _, dep := range deps {
			// Avoid self dependencies, just a safety net.
			if dep != app.Name {
				installed, found := installedApps[dep]
				if !found || !installed {
					dependenciesNotInstalled = append(dependenciesNotInstalled, dep)
				}
			}
		}
	}

	if len(dependenciesNotInstalled) > 0 {
		r.logger.Debugf(ctx, "Not creating chart for app %q: dependencies not satisfied %v", app.Name, dependenciesNotInstalled)
		return dependenciesNotInstalled, nil
	}

	r.logger.Debugf(ctx, "Dependencies for App %q are satisfied", app.Name)

	return nil, nil
}

// pickHelmRepository finds the primary repository that is placed first on
// the list of repositories to check for an app inside. It follows the below rules:
//  1. If Catalog's .spec.repositories list is empty, use the Catalog's .spec.storage item.
//  2. If Catalog's .spec.repositories list is not empty, and contains a single item,
//     use this item.
//  3. If Catalog's .spec.repository list is not empty, and contains more than one item,
//     move on to the next steps to determine the repository.
//  4. If HelmRelease of given name and namespace is not found, use the first item
//     from the Catalog's .spec.repositories list.
//  5. If HelmRelease of given name and namespace is found, but HelmRepository referenced
//     does not match any of the Catalog repositories, use the first one from the
//     Catalog's .spec.repositories list.
//  6. If HelmRelease of given name and namespace is found, and HelmRepository referenced
//     matches one of the Catalog's repositories, but artifact failure is reported, use
//     next item from Catalog's repositories list.
//  7. If HelmRelease of given name and namespace is found, and HelmRepository referenced
//     matches one of the Catalog's repositories, and no artifact failure is reported, re-use
//     this repository.
func (r *Resource) pickHelmRepository(ctx context.Context, cc *controllercontext.Context, cr v1alpha1.App) (helmRepository, error) {
	switch len(cc.Catalog.Spec.Repositories) {
	case 0:
		return helmRepository{cc.Catalog.Spec.Storage.Type, cc.Catalog.Spec.Storage.URL}, nil
	case 1:
		return helmRepository{cc.Catalog.Spec.Repositories[0].Type, cc.Catalog.Spec.Repositories[0].URL}, nil
	}

	var helmRelease helmv2.HelmRelease
	err := cc.Clients.K8s.CtrlClient().Get(
		ctx,
		types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace},
		&helmRelease,
	)
	if apierrors.IsNotFound(err) {
		// Repositories is guaranteed by Custom Resource Definition to have at least one entry.
		return helmRepository{cc.Catalog.Spec.Repositories[0].Type, cc.Catalog.Spec.Repositories[0].URL}, nil
	} else if err != nil {
		return helmRepository{}, microerror.Mask(err)
	}

	// Check currently selected repository
	repositoryIndex := -1
	for i, repo := range cc.Catalog.Spec.Repositories {
		name, err := appopkey.GetHelmRepositoryName(cc.Catalog.Name, repo.Type, repo.URL)
		if err != nil {
			return helmRepository{}, microerror.Mask(err)
		}

		if name == helmRelease.Spec.Chart.Spec.SourceRef.Name {
			repositoryIndex = i
			break
		}
	}

	if repositoryIndex == -1 {
		// Could not match current HelmRepository name to any existing HelmRepositories.
		// Maybe the list was updated. Let's pick any existing repository.
		r.logger.Debugf(
			ctx,
			"could not match HelmRepository %q to any of %q Catalog repositories; using default",
			helmRelease.Spec.Chart.Spec.SourceRef.Name,
			cc.Catalog.Name,
		)
		return helmRepository{cc.Catalog.Spec.Repositories[0].Type, cc.Catalog.Spec.Repositories[0].URL}, nil
	}

	condition := apimeta.FindStatusCondition(helmRelease.Status.Conditions, fluxmeta.ReadyCondition)
	if condition != nil && condition.Reason == helmv2.ArtifactFailedReason {
		// ArtifactFailedReason condition has been observed as the latest condition on the
		// HelmRelease CR indicating problems with the Helm Chart for an app.
		repositoryIndex = (repositoryIndex + 1) % len(cc.Catalog.Spec.Repositories)
	}
	return helmRepository{cc.Catalog.Spec.Repositories[repositoryIndex].Type, cc.Catalog.Spec.Repositories[repositoryIndex].URL}, nil
}

// buildTarballURL builds a tarball URL and returns error if it does not point to an existing
// resource in a given Helm repository. The URL is not returned as it is not needed for the
// HelmRelease CR configuration.
func (r *Resource) buildTarballURL(ctx context.Context, cc *controllercontext.Context, cr v1alpha1.App, repository helmRepository) (err error) {
	if key.CatalogVisibility(cc.Catalog) == "internal" || repository.repoType == "oci" {
		// For internal catalogs we generate the URL as its predictable
		// and to avoid having chicken and egg problems.
		// For OCI repositories there is no discovery mechanism, so we just
		// make an assumption about URL format.
		_, err = appcatalog.NewTarballURL(repository.repoURL, key.AppName(cr), key.Version(cr))
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	// For all other catalogs we check the index.yaml for compatibility
	// with community catalogs.
	index, err := r.indexCache.GetIndex(ctx, repository.repoURL)
	if err != nil {
		r.logger.Errorf(ctx, err, "failed to get index.yaml from %q", repository.repoURL)
	}
	if index == nil {
		return microerror.Maskf(indexNotFoundError, "index %#v for %q is <nil>", index, repository.repoURL)
	}
	if len(index.Entries) == 0 {
		return microerror.Maskf(catalogEmptyError, "index %#v for %q has no entries", index, repository.repoURL)
	}

	entries, ok := index.Entries[cr.Spec.Name]
	if !ok {
		return microerror.Maskf(appNotFoundError, "no entries for app %#q in index.yaml for %q", cr.Spec.Name, repository.repoURL)
	}

	// We first try with the full version set in .spec.version of the app CR.
	url, err := getEntryURL(entries, cr.Spec.Name, cr.Spec.Version)
	if err != nil {
		// We try again without the `v` prefix. This enables us to use the
		// Flux Image Automation controller to automatically update apps.
		url, err = getEntryURL(
			entries,
			cr.Spec.Name,
			strings.TrimPrefix(cr.Spec.Version, "v"),
		)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if url == "" {
		return microerror.Maskf(appVersionNotFoundError, "found entry for app %#q but URL is not specified", cr.Spec.Name)
	}

	if !isValidURL(url) {
		// URL may be relative. If so we join it to the Catalog Storage URL.
		_, err = joinRelativeURL(repository.repoURL, url)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return err
}

// fallbackRepositories returns list of repositories omitting the one that
// suppose to be already configured.
func fallbackRepositories(catalog v1alpha1.Catalog, repository helmRepository) []helmRepository {
	repositories := []helmRepository{}

	for _, repo := range catalog.Spec.Repositories {
		if repo.URL == repository.repoURL && repo.Type == repository.repoType {
			continue
		}
		repositories = append(repositories, helmRepository{repo.Type, repo.URL})
	}

	return repositories
}

// generateConfig checks for values ConfigMap and Secret and return a slice acceptable
// by the HelmRelease CR.
func generateConfig(ctx context.Context, k8sClient kubernetes.Interface, cr v1alpha1.App, catalog v1alpha1.Catalog) ([]helmv2.ValuesReference, map[string]string, error) {
	config := make([]helmv2.ValuesReference, 0)
	version := map[string]string{}

	if hasConfigMap(cr, catalog) {
		configMapName := key.ChartConfigMapName(cr)
		cm, err := k8sClient.CoreV1().ConfigMaps(cr.Namespace).Get(ctx, configMapName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// no-op
		} else if err != nil {
			return []helmv2.ValuesReference{}, map[string]string{}, microerror.Mask(err)
		} else {
			config = append(config, helmv2.ValuesReference{
				Kind: "ConfigMap",
				Name: configMapName,
			})
			version[annotation.AppOperatorLatestConfigMapVersion] = cm.GetResourceVersion()
		}
	}

	if hasSecret(cr, catalog) {
		secretName := key.ChartSecretName(cr)
		secret, err := k8sClient.CoreV1().Secrets(cr.Namespace).Get(ctx, secretName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// no-op
		} else if err != nil {
			return []helmv2.ValuesReference{}, map[string]string{}, microerror.Mask(err)
		} else {
			config = append(config, helmv2.ValuesReference{
				Kind: "Secret",
				Name: secretName,
			})
			version[annotation.AppOperatorLatestSecretVersion] = secret.GetResourceVersion()
		}
	}

	return config, version, nil
}

func generateInstall(cr v1alpha1.App) *helmv2.Install {
	install := helmv2.Install{}

	if key.InstallSkipCRDs(cr) {
		install.SkipCRDs = true
	}

	timeout := key.InstallTimeout(cr)
	if timeout != nil {
		install.Timeout = timeout
	}

	// This is to replace the namespace resource of Chart Operator.
	// Now Helm Controller is going to be in charge of creating namespace.
	install.CreateNamespace = true

	// This is to satisfy options previously configured by
	// the Chart Operator and helmclient package.
	// Note: the wait-related options are not disabled here, because
	// Helm Controller, unlike the Chart Operator, is not capable of
	// checking the actual release state, hence the only way for it
	// to really know the status, is to wait for Helm actions to report it.
	install.DisableOpenAPIValidation = true

	return &install
}

func generateRollback(cr v1alpha1.App) *helmv2.Rollback {
	rollback := helmv2.Rollback{}

	timeout := key.RollbackTimeout(cr)
	if timeout != nil {
		rollback.Timeout = timeout
	}

	return &rollback
}

func generateUninstall(cr v1alpha1.App) *helmv2.Uninstall {
	uninstall := helmv2.Uninstall{}

	timeout := key.UninstallTimeout(cr)
	if timeout != nil {
		uninstall.Timeout = timeout
	}

	uninstall.DeletionPropagation = ptr.To("background")

	return &uninstall
}

func generateUpgrade(cr v1alpha1.App) *helmv2.Upgrade {
	upgrade := helmv2.Upgrade{}

	timeout := key.UpgradeTimeout(cr)
	if timeout != nil {
		upgrade.Timeout = timeout
	}

	upgrade.DisableOpenAPIValidation = false
	upgrade.Force = false

	return &upgrade
}

func getDependenciesFromCR(app v1alpha1.App) ([]string, error) {
	deps := make([]string, 0)
	dependsOn, found := app.Annotations[annotationAppDependsOn]
	if found {
		deps = strings.Split(dependsOn, ",")
	}

	ret := make([]string, 0)
	for _, dep := range deps {
		if dep != "" {
			ret = append(ret, dep)
		}
	}

	return ret, nil
}

func getEntryURL(entries []indexcache.Entry, app, version string) (string, error) {
	for _, e := range entries {
		if e.Version == version {
			if len(e.Urls) == 0 {
				return "", microerror.Maskf(appVersionNotFoundError, "no URL in index.yaml for app %#q version %#q", app, version)
			}

			return e.Urls[0], nil
		}
	}

	return "", microerror.Maskf(appVersionNotFoundError, "no app %#q in index.yaml with given version %#q", app, version)
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

// processLabels ensures the giantswarm.io/managed-by label is accurate.
//
// Any other labels added to the app custom resource are passed on to the HelmRelease
// custom resource. This includes the version label which is used as a selector in the
// status watcher.
func processLabels(projectName string, inputLabels map[string]string) map[string]string {
	// These labels are required.
	labels := map[string]string{
		label.ManagedBy: projectName,
	}

	for k, v := range inputLabels {
		// All other labels, except for old managed-by are simply
		// passed down to the HelmRelease CR.
		if k != label.ManagedBy {
			labels[k] = v
		}
	}

	return labels
}

func setStatus(cc *controllercontext.Context, err error) {
	switch microerror.Cause(err) {
	case appNotFoundError:
		addStatusToContext(cc, err.Error(), status.AppNotFoundStatus)
	case appVersionNotFoundError:
		addStatusToContext(cc, err.Error(), status.AppVersionNotFoundStatus)
	case catalogEmptyError:
		addStatusToContext(cc, err.Error(), status.CatalogEmptyStatus)
	case indexNotFoundError:
		addStatusToContext(cc, err.Error(), status.IndexNotFoundStatus)
	default:
		addStatusToContext(cc, err.Error(), status.UnknownError)
	}
}
