package chartoperator

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/app/v5/pkg/values"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
)

const (
	Name = "chartoperator"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	FileSystem afero.Fs
	G8sClient  versioned.Interface
	K8sClient  kubernetes.Interface
	Logger     micrologger.Logger
	Values     *values.Values

	// Settings.
	ChartNamespace string
}

type Resource struct {
	// Dependencies.
	fileSystem afero.Fs
	g8sClient  versioned.Interface
	k8sClient  kubernetes.Interface
	logger     micrologger.Logger
	values     *values.Values

	// Settings.
	chartNamespace string
}

// New creates a new configured chartoperator resource.
func New(config Config) (*Resource, error) {
	if config.FileSystem == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.FileSystem must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Values == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Values must not be empty", config)
	}

	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		fileSystem: config.FileSystem,
		g8sClient:  config.G8sClient,
		k8sClient:  config.K8sClient,
		logger:     config.Logger,
		values:     config.Values,

		chartNamespace: config.ChartNamespace,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

func (r Resource) installChartOperator(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	chartOperatorValues, err := r.values.MergeAll(ctx, cr, cc.Catalog)
	if err != nil {
		return microerror.Mask(err)
	}

	// check app CR for chart-operator and fetching app-catalog name and version.
	var tarballURL string
	{
		tarballURL, err = appcatalog.NewTarballURL(key.CatalogStorageURL(cc.Catalog), key.AppName(cr), key.Version(cr))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var tarballPath string
	{
		tarballPath, err = cc.Clients.Helm.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		defer func() {
			err := r.fileSystem.Remove(tarballPath)
			if err != nil {
				r.logger.Errorf(ctx, err, "deletion of %#q failed", tarballPath)
			}
		}()
	}

	{
		opts := helmclient.InstallOptions{
			ReleaseName: cr.Name,
		}
		err = cc.Clients.Helm.InstallReleaseFromTarball(ctx,
			tarballPath,
			key.Namespace(cr),
			chartOperatorValues,
			opts)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r Resource) updateChartOperator(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	chartOperatorValues, err := r.values.MergeAll(ctx, cr, cc.Catalog)
	if err != nil {
		return microerror.Mask(err)
	}

	// check app CR for chart-operator and fetching app-catalog name and version.
	var tarballURL string
	{
		tarballURL, err = appcatalog.NewTarballURL(key.CatalogStorageURL(cc.Catalog), key.AppName(cr), key.Version(cr))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var tarballPath string
	{
		tarballPath, err = cc.Clients.Helm.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		defer func() {
			err := r.fileSystem.Remove(tarballPath)
			if err != nil {
				r.logger.Errorf(ctx, err, "deletion of %#q failed", tarballPath)
			}
		}()
	}

	{
		opts := helmclient.UpdateOptions{
			Force: true,
		}
		err = cc.Clients.Helm.UpdateReleaseFromTarball(ctx,
			tarballPath,
			key.Namespace(cr),
			cr.Name,
			chartOperatorValues,
			opts)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r Resource) uninstallChartOperator(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = cc.Clients.Helm.DeleteRelease(ctx, key.Namespace(cr), cr.Name)
	if helmclient.IsReleaseNotFound(err) {
		// no-op
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r Resource) deleteFinalizers(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	chart, err := cc.Clients.K8s.G8sClient().ApplicationV1alpha1().Charts(r.chartNamespace).Get(ctx, cr.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// no-op
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if len(chart.GetFinalizers()) > 0 {
		r.logger.Debugf(ctx, "deleting remaining finalizers on %#q", key.AppName(cr))

		chart.Finalizers = nil

		_, err := cc.Clients.K8s.G8sClient().ApplicationV1alpha1().Charts(r.chartNamespace).Update(ctx, chart, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "deleted remaining finalizers on %#q", key.AppName(cr))
	}

	return nil
}
