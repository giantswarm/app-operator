package chartoperator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/app/v7/pkg/values"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

const (
	Name = "chartoperator"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	FileSystem afero.Fs
	CtrlClient client.Client
	K8sClient  kubernetes.Interface
	Logger     micrologger.Logger
	Values     *values.Values

	// Settings.
	ChartNamespace    string
	WorkloadClusterID string
}

type Resource struct {
	// Dependencies.
	fileSystem afero.Fs
	ctrlClient client.Client
	k8sClient  kubernetes.Interface
	logger     micrologger.Logger
	values     *values.Values

	// Settings.
	chartNamespace    string
	workloadClusterID string
}

// New creates a new configured chartoperator resource.
func New(config Config) (*Resource, error) {
	if config.FileSystem == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.FileSystem must not be empty", config)
	}
	if config.CtrlClient == client.Client(nil) {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.K8sClient == kubernetes.Interface(nil) {
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
		ctrlClient: config.CtrlClient,
		k8sClient:  config.K8sClient,
		logger:     config.Logger,
		values:     config.Values,

		chartNamespace:    config.ChartNamespace,
		workloadClusterID: config.WorkloadClusterID,
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
			ReleaseName: key.AppName(cr),
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

// The `triggerReconciliation` gets executed upon successfull installation of
// the chart-operator in the workload cluster. It checks for App CRs without
// corresponding Chart CRs in the workload cluster, and then annotates them
// to trigger reconciliation and speed up bootstrapping.
func (r Resource) triggerReconciliation(ctx context.Context, chartOperatorApp v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	// Find all the Apps CR for a given cluster.
	// If `operatorApp` is an org-namespaced App use the cluster label selector.
	var appList v1alpha1.AppList
	{
		o := client.ListOptions{}

		var selector k8slabels.Selector

		if key.IsInOrgNamespace(chartOperatorApp) {
			selector, err = k8slabels.Parse(fmt.Sprintf("%s=%s", label.Cluster, key.ClusterLabel(chartOperatorApp)))
			if err != nil {
				return microerror.Mask(err)
			}

			o.LabelSelector = selector
		}

		err = r.ctrlClient.List(ctx, &appList, &o)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// For each App, check if the corresponding Chart CR exists.
	// If not, annotate the App to trigger the reconciliation.
	for i, app := range appList.Items {
		// Skip for in-cluster apps and the chart-operator app itself.
		if key.InCluster(app) || app.ObjectMeta.Name == chartOperatorApp.ObjectMeta.Name {
			continue
		}

		name := key.ChartName(app, r.workloadClusterID)

		var chart v1alpha1.Chart
		err = cc.Clients.K8s.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: name, Namespace: r.chartNamespace},
			&chart,
		)

		// if chart CR is not found, trigger sync
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "did not find chart %#q in namespace %#q", name, r.chartNamespace)
			r.logger.Debugf(ctx, "annotating %#q app", app.ObjectMeta.Name)

			if len(app.GetAnnotations()) == 0 {
				app.Annotations = map[string]string{}
			}

			bytes, err := json.Marshal(map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]string{
						annotation.AppOperatorTriggerReconciliation: metav1.Now().Format(time.RFC3339),
					},
				},
			})
			if err != nil {
				return microerror.Mask(err)
			}

			// Indexing is used to fix the `G601: Implicit memory aliasing in for loop`
			err = r.ctrlClient.Patch(ctx, &appList.Items[i], client.RawPatch(types.MergePatchType, bytes))
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.Debugf(ctx, "annotated %#q app", app.ObjectMeta.Name)
		} else if err != nil {
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
			key.AppName(cr),
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

	err = cc.Clients.Helm.DeleteRelease(ctx, key.Namespace(cr), key.AppName(cr), helmclient.DeleteOptions{})
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

	var chart v1alpha1.Chart

	err = cc.Clients.K8s.CtrlClient().Get(
		ctx,
		types.NamespacedName{Name: key.AppName(cr), Namespace: r.chartNamespace},
		&chart,
	)
	if apierrors.IsNotFound(err) {
		// no-op
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if len(chart.GetFinalizers()) > 0 {
		r.logger.Debugf(ctx, "deleting remaining finalizers on %#q", key.AppName(cr))

		chart.Finalizers = nil

		err = cc.Clients.K8s.CtrlClient().Update(ctx, &chart)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "deleted remaining finalizers on %#q", key.AppName(cr))
	}

	return nil
}
