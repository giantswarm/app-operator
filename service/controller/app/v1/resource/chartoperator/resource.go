package chartoperator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
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

	chartOperatorValues, err := r.mergeChartOperatorValues(ctx, cr, cc.AppCatalog)
	if err != nil {
		return microerror.Mask(err)
	}

	// check app CR for chart-operator and fetching app-catalog name and version.
	var tarballURL string
	{
		tarballURL, err = appcatalog.NewTarballURL(key.AppCatalogStorageURL(cc.AppCatalog), key.AppName(cr), key.Version(cr))
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
				r.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", tarballPath), "stack", fmt.Sprintf("%#v", err))
			}
		}()
	}

	{
		err = cc.Clients.Helm.InstallReleaseFromTarball(ctx, tarballPath, key.Namespace(cr), helm.ReleaseName(cr.Name), helm.ValueOverrides(chartOperatorValues))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		// We wait for the chart-operator deployment to be ready so the
		// chart CRD is installed. This allows the chart
		// resource to create CRs in the same reconcilation loop.
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for ready %#q deployment", cr.Name))

		o := func() error {
			err := r.checkDeploymentReady(ctx, cc.Clients.K8s, cr)
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
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%#q deployment is not ready retrying in %s", cr.Name, delay), "stack", fmt.Sprintf("%#v", err))
		}

		err = backoff.RetryNotify(o, b, n)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%#q deployment is ready", cr.Name))
	}

	return nil
}

func (r Resource) updateChartOperator(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	chartOperatorValues, err := r.mergeChartOperatorValues(ctx, cr, cc.AppCatalog)
	if err != nil {
		return microerror.Mask(err)
	}

	// check app CR for chart-operator and fetching app-catalog name and version.
	var tarballURL string
	{
		tarballURL, err = appcatalog.NewTarballURL(key.AppCatalogStorageURL(cc.AppCatalog), key.AppName(cr), key.Version(cr))
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
				r.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", tarballPath), "stack", fmt.Sprintf("%#v", err))
			}
		}()
	}

	{
		err = cc.Clients.Helm.UpdateReleaseFromTarball(ctx, cr.Name, tarballPath, helm.UpdateValueOverrides(chartOperatorValues), helm.UpgradeForce(true))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for ready %#q deployment", cr.Name))

		o := func() error {
			err := r.checkDeploymentReady(ctx, cc.Clients.K8s, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}

		b := backoff.NewConstant(20*time.Second, 10*time.Second)
		n := func(err error, delay time.Duration) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%#q deployment is not ready retrying in %s", cr.Name, delay), "stack", fmt.Sprintf("%#v", err))
		}

		err = backoff.RetryNotify(o, b, n)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%#q deployment is ready", cr.Name))
	}

	return nil
}

func (r *Resource) mergeChartOperatorValues(ctx context.Context, cr v1alpha1.App, catalog v1alpha1.AppCatalog) ([]byte, error) {
	var chartOperatorValues []byte
	{
		values, err := r.values.MergeAll(ctx, cr, catalog)
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

// checkDeploymentReady checks for the specified deployment that the number of
// ready replicas matches the desired state.
func (r *Resource) checkDeploymentReady(ctx context.Context, k8sClient kubernetes.Interface, cr v1alpha1.App) error {
	namespace := key.Namespace(cr)
	deploy, err := k8sClient.AppsV1().Deployments(namespace).Get(cr.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return microerror.Maskf(notReadyError, "deployment %#q not found", cr.Name)
	} else if err != nil {
		return microerror.Mask(err)
	}

	if deploy.Status.ReadyReplicas != *deploy.Spec.Replicas {
		return microerror.Maskf(notReadyError, "deployment %#q want %d replicas %d ready", cr.Name, *deploy.Spec.Replicas, deploy.Status.ReadyReplicas)
	}

	// Deployment is ready.
	return nil
}
