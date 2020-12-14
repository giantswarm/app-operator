package releasemigration

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	found, err := findMigrationApp(ctx, helmClient, tillerNamespace)
	if err != nil {
		return microerror.Mask(err)
	}

	if !found {
		// no-op
		return nil
	}

	err = helmClient.DeleteRelease(ctx, tillerNamespace, migrationApp)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Resource) ensureReleasesMigrated(ctx context.Context, ctrlClient client.Client, helmClient helmclient.Interface, tillerNamespace string) error {
	// Found all dangling helm release v2
	releases, err := r.findHelmV2Releases(ctx, ctrlClient, tillerNamespace)
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
					r.logger.Errorf(ctx, err, "deletion of %#q failed", tarballPath)
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
		completed, err := checkMigrationJobStatus(ctx, ctrlClient, "giantswarm")
		if err != nil {
			return microerror.Mask(err)
		}

		if !completed {
			releases, err := r.findHelmV2Releases(ctx, ctrlClient, tillerNamespace)
			if err != nil {
				return microerror.Mask(err)
			}

			desc := fmt.Sprintf("%d helm v2 releases not migrated", len(releases))
			r.logger.Debugf(ctx, desc)

			return microerror.Maskf(executionFailedError, desc)
		}
		r.logger.Debugf(ctx, "migration completed")

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Debugf(ctx, "migration not complete")
	}

	b := backoff.NewConstant(20*time.Minute, 10*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) findHelmV2Releases(ctx context.Context, ctrlClient client.Client, tillerNamespace string) ([]string, error) {
	chartMap, err := getChartMap(ctx, ctrlClient, r.chartNamespace)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cms := &corev1.ConfigMapList{}
	lo := client.ListOptions{
		Namespace:     tillerNamespace,
		LabelSelector: fmt.Sprintf("%s=%s", "OWNER", "TILLER"),
	}

	// Check whether helm 2 release configMaps still exist.
	err = ctrlClient.List(ctx, lo)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	hasReleases := map[string]bool{}
	for _, cm := range cms.Items {
		name := cm.GetLabels()["NAME"]

		// Skip Helm release if it has no matching chart CR.
		if _, ok := chartMap[name]; !ok {
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

func findMigrationApp(ctx context.Context, helmClient helmclient.Interface, tillerNamespace string) (bool, error) {
	_, err := helmClient.GetReleaseContent(ctx, tillerNamespace, migrationApp)
	if helmclient.IsReleaseNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}
	return true, nil
}
