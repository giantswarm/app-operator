//go:build k8srequired
// +build k8srequired

package setup

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/app-operator/v6/integration/key"
	"github.com/giantswarm/app-operator/v6/integration/templates"
	"github.com/giantswarm/app-operator/v6/pkg/project"
)

type appConfiguration struct {
	appName      string
	appNamespace string
	appValues    string
	appVersion   string
	catalogURL   string
}

func Setup(m *testing.M, config Config) {
	ctx := context.Background()

	var v int
	var err error

	err = installResources(ctx, config)
	if err != nil {
		config.Logger.Errorf(ctx, err, "failed to install app-operator dependent resources")
		v = 1
	}

	if v == 0 {
		if err != nil {
			config.Logger.Errorf(ctx, err, "failed to create operator resources")
			v = 1
		}
	}

	if v == 0 {
		v = m.Run()
	}

	os.Exit(v)
}

func installResources(ctx context.Context, config Config) error {
	var err error

	{
		err = config.K8s.EnsureNamespaceCreated(ctx, key.GiantSwarmNamespace())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = config.K8s.EnsureNamespaceCreated(ctx, key.FluxSystemNamespace())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	apps := []appConfiguration{
		appConfiguration{
			appName:      project.Name(),
			appNamespace: key.GiantSwarmNamespace(),
			appValues:    templates.AppOperatorVintageValues,
			appVersion:   key.AppOperatorInTestVersion(),
			catalogURL:   key.ControlPlaneTestCatalogStorageURL(),
		},
	}

	if config.HelmControllerBackend {
		apps[0].appValues = templates.AppOperatorCAPIValues
	}

	if config.HelmControllerBackend {
		apps = append(
			apps,
			appConfiguration{
				appName:      key.FluxAppName(),
				appNamespace: key.FluxSystemNamespace(),
				appValues:    "",
				appVersion:   key.FluxAppVersion(),
				catalogURL:   key.StableCatalogStorageHelmURL(),
			},
		)
	}

	for _, app := range apps {
		var tarballURL string
		{
			config.Logger.Debugf(ctx, "getting %#q tarball URL", app.appName)

			o := func() error {
				tarballURL, err = appcatalog.GetLatestChart(ctx, app.catalogURL, app.appName, app.appVersion)
				if err != nil {
					return microerror.Mask(err)
				}

				return nil
			}

			b := backoff.NewConstant(5*time.Minute, 10*time.Second)
			n := backoff.NewNotifier(config.Logger, ctx)

			err = backoff.RetryNotify(o, b, n)
			if err != nil {
				return microerror.Mask(err)
			}

			config.Logger.Debugf(ctx, "tarball URL is %#q", tarballURL)
		}

		var tarballPath string
		{
			config.Logger.Debugf(ctx, "pulling tarball")

			tarballPath, err = config.HelmClient.PullChartTarball(ctx, tarballURL)
			if err != nil {
				return microerror.Mask(err)
			}

			config.Logger.Debugf(ctx, "tarball path is %#q", tarballPath)
		}

		var values map[string]interface{}
		{
			err = yaml.Unmarshal([]byte(app.appValues), &values)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		{
			defer func() {
				fs := afero.NewOsFs()
				err := fs.Remove(tarballPath)
				if err != nil {
					config.Logger.Errorf(ctx, err, "deletion of %#q failed", tarballPath)
				}
			}()

			config.Logger.Debugf(ctx, "installing %#q", app.appName)

			// Release is named app-operator-unique as some functionality is only
			// implemented for the unique instance.
			opts := helmclient.InstallOptions{
				ReleaseName: app.appName,
				Wait:        true,
			}
			err = config.HelmClient.InstallReleaseFromTarball(ctx,
				tarballPath,
				app.appNamespace,
				values,
				opts)
			if err != nil {
				return microerror.Mask(err)
			}

			config.Logger.Debugf(ctx, "installed %#q", app.appVersion)
		}
	}

	return nil
}
