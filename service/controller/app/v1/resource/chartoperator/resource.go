package chartoperator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"github.com/spf13/afero"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	"github.com/giantswarm/app-operator/service/controller/app/v1/values"
)

const (
	Name = "chartoperatorv1"
)

const (
	namespace = "giantswarm"
	release   = "chart-operator"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	FileSystem afero.Fs
	G8sClient  versioned.Interface
	K8sClient  kubernetes.Interface
	Logger     micrologger.Logger
	Values     *values.Values
}

type Resource struct {
	// Dependencies.
	fileSystem afero.Fs
	g8sClient  versioned.Interface
	k8sClient  kubernetes.Interface
	logger     micrologger.Logger
	values     *values.Values

	// Settings.
	registryDomain string
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

	r := &Resource{
		// Dependencies.
		fileSystem: config.FileSystem,
		g8sClient:  config.G8sClient,
		k8sClient:  config.K8sClient,
		logger:     config.Logger,
		values:     config.Values,
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

	chartOperatorAppCR, err := r.getChartOperatorAppCR(ctx, cr.Namespace)
	if err != nil {
		return microerror.Mask(err)
	}

	chartOperatorValues, err := r.mergeChartOperatorValues(ctx, chartOperatorAppCR, cc.AppCatalog)
	if err != nil {
		return microerror.Mask(err)
	}

	// check app CR for chart-operator and fetching app-catalog name and version.
	var tarballURL string
	{
		tarballURL, err = appcatalog.NewTarballURL(key.AppCatalogStorageURL(cc.AppCatalog), release, key.Version(*chartOperatorAppCR))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var tarballPath string
	{
		tarballPath, err = cc.HelmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		defer func() {
			err := r.fileSystem.Remove(tarballPath)
			if err != nil {
				r.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", tarballPath), "stack", fmt.Sprintf("%#v", err))
			}
		}()
	}

	{
		err = cc.HelmClient.InstallReleaseFromTarball(ctx, tarballPath, namespace, helm.ReleaseName(release), helm.ValueOverrides(chartOperatorValues))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		// We wait for the chart-operator deployment to be ready so the
		// chart CRD is installed. This allows the chart
		// resource to create CRs in the same reconcilation loop.
		r.logger.LogCtx(ctx, "level", "debug", "message", "waiting for ready chart-operator deployment")

		o := func() error {
			err := r.checkDeploymentReady(ctx, cc.K8sClient)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}

		// Wait for chart-operator to be deployed. If it takes longer than
		// the timeout the chartconfig CRs will be created during the next
		// reconciliation loop.
		b := backoff.NewConstant(20*time.Second, 5*time.Second)
		n := func(err error, delay time.Duration) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%#q deployment is not ready retrying in %s", release, delay), "stack", fmt.Sprintf("%#v", err))
		}

		err = backoff.RetryNotify(o, b, n)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "chart-operator deployment is ready")
	}

	return nil
}

func (r Resource) updateChartOperator(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	chartOperatorAppCR, err := r.getChartOperatorAppCR(ctx, cr.Namespace)
	if err != nil {
		return microerror.Mask(err)
	}

	chartOperatorValues, err := r.mergeChartOperatorValues(ctx, chartOperatorAppCR, cc.AppCatalog)
	if err != nil {
		return microerror.Mask(err)
	}

	// check app CR for chart-operator and fetching app-catalog name and version.
	var tarballURL string
	{
		tarballURL, err = appcatalog.NewTarballURL(key.AppCatalogStorageURL(cc.AppCatalog), release, key.Version(*chartOperatorAppCR))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var tarballPath string
	{
		tarballPath, err = cc.HelmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		defer func() {
			err := r.fileSystem.Remove(tarballPath)
			if err != nil {
				r.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", tarballPath), "stack", fmt.Sprintf("%#v", err))
			}
		}()
	}

	{
		err = cc.HelmClient.UpdateReleaseFromTarball(ctx, release, tarballPath, helm.UpdateValueOverrides(chartOperatorValues), helm.UpgradeForce(true))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) getChartOperatorAppCR(ctx context.Context, namespace string) (*v1alpha1.App, error) {
	var chartOperatorAppCR *v1alpha1.App
	var err error
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "finding chart-operator app CR")

		chartOperatorAppCR, err = r.g8sClient.ApplicationV1alpha1().Apps(namespace).Get(release, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "can't find chart-operator app CR")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling the reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "found chart-operator app CR")
	}
	return chartOperatorAppCR, nil
}

func (r *Resource) mergeChartOperatorValues(ctx context.Context, cr *v1alpha1.App, catalog v1alpha1.AppCatalog) ([]byte, error) {
	var chartOperatorValues []byte
	{
		values, err := r.values.MergeAll(ctx, *cr, catalog)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		chartOperatorValues, err = json.Marshal(values)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	return chartOperatorValues, nil
}
