package releasemigration

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/key"
)

const (
	Name = "releasemigrationv1"

	migrationApp = "helm-2to3-migration"
)

type Config struct {
	// Dependencies.
	Logger micrologger.Logger

	// Settings.
	ChartNamespace string
	ImageRegistry  string
}

type Resource struct {
	// Dependencies.
	logger micrologger.Logger

	// Settings.
	chartNamespace string
	imageRegistry  string
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}
	if config.ImageRegistry == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ImageRegistry must not be empty", config)
	}

	r := &Resource{
		logger: config.Logger,

		chartNamespace: config.ChartNamespace,
		imageRegistry:  config.ImageRegistry,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) deleteMigrationApp(ctx context.Context, helmClient helmclient.Interface, tillerNamespace string) error {
	err := helmClient.DeleteRelease(ctx, tillerNamespace, migrationApp)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Resource) ensureReleasesMigrated(ctx context.Context, k8sClient k8sclient.Interface, helmClient helmclient.Interface, tillerNamespace string) error {
	// Found all dangling helm release v2
	releases, err := r.findHelmV2Releases(k8sClient, tillerNamespace)
	if err != nil {
		return microerror.Mask(err)
	}

	// Install helm-2to3-migration app
	{
		var tarballPath string
		{
			tarballURL, err := appcatalog.GetLatestChart(ctx, key.DefaultCatalogStorageURL(), "helm-2to3-migration", "")
			if err != nil {
				return microerror.Mask(err)
			}

			tarballPath, err = helmClient.PullChartTarball(ctx, tarballURL)
			if err != nil {
				return microerror.Mask(err)
			}

			defer func() {
				fs := afero.NewOsFs()
				err := fs.Remove(tarballPath)
				if err != nil {
					r.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", tarballPath), "stack", fmt.Sprintf("%#v", err))
				}
			}()

			opts := helmclient.InstallOptions{
				ReleaseName: migrationApp,
			}

			values := map[string]interface{}{
				"image": map[string]interface{}{
					"registry": r.imageRegistry,
				},
				"releases": releases,
				"tiller": map[string]interface{}{
					"namespace": tillerNamespace,
				},
			}

			err = helmClient.InstallReleaseFromTarball(ctx, tarballPath, tillerNamespace, values, opts)
			if helmclient.IsReleaseAlreadyExists(err) {
				return microerror.Maskf(releaseAlreadyExistsError, "release %#q already exists", migrationApp)
			} else if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	// Wait until all helm v2 release are deleted
	o := func() error {
		releases, err := r.findHelmV2Releases(k8sClient, tillerNamespace)
		if err != nil {
			return microerror.Mask(err)
		}
		if len(releases) > 0 {
			desc := fmt.Sprintf("%d helm v2 releases not migrated", len(releases))
			r.logger.Log("level", "debug", "message", desc)

			return microerror.Maskf(executionFailedError, desc)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", "migration not complete")
	}

	b := backoff.NewConstant(5*time.Minute, 10*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) findHelmV2Releases(k8sClient k8sclient.Interface, tillerNamespace string) ([]string, error) {
	charts := make(map[string]bool)

	// Get list of chart CRs as not all helm 2 releases will have a chart CR.
	list, err := k8sClient.G8sClient().ApplicationV1alpha1().Charts(r.chartNamespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for _, chart := range list.Items {
		charts[chart.Name] = true
	}

	lo := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", "OWNER", "TILLER"),
	}

	// Check whether helm 2 release configMaps still exist.
	cms, err := k8sClient.K8sClient().CoreV1().ConfigMaps(tillerNamespace).List(lo)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	hasReleases := map[string]bool{}
	for _, cm := range cms.Items {
		name := cm.GetLabels()["NAME"]

		// Skip Helm release if it has no matching chart CR.
		if _, ok := charts[name]; !ok {
			continue
		}

		if _, ok := hasReleases[name]; !ok {
			hasReleases[name] = true
		}
	}

	releases := make([]string, 0, len(hasReleases))
	for k := range hasReleases {
		releases = append(releases, k)
	}

	return releases, nil
}
