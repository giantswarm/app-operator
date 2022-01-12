package chartstatus

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

var chartResource = schema.GroupVersionResource{Group: "application.giantswarm.io", Version: "v1alpha1", Resource: "charts"}

type ChartStatusWatcherConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	ChartNamespace    string
	UniqueApp         bool
	WatchNamespace    string
	WorkloadClusterID string
}

type ChartStatusWatcher struct {
	k8sClient  k8sclient.Interface
	kubeConfig kubeconfig.Interface
	logger     micrologger.Logger

	chartNamespace    string
	uniqueApp         bool
	watchNamespace    string
	workloadClusterID string
}

func NewChartStatusWatcher(config ChartStatusWatcherConfig) (*ChartStatusWatcher, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}
	if config.WatchNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.WatchNamespace must not be empty", config)
	}
	if config.WorkloadClusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.WorkloadClusterID must not be empty", config)
	}

	var kubeConfig kubeconfig.Interface
	var err error
	{
		c := kubeconfig.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		kubeConfig, err = kubeconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &ChartStatusWatcher{
		k8sClient:  config.K8sClient,
		kubeConfig: kubeConfig,
		logger:     config.Logger,

		// We get a kubeconfig for the cluster from the chart-operator app CR.
		chartNamespace:    config.ChartNamespace,
		uniqueApp:         config.UniqueApp,
		watchNamespace:    config.WatchNamespace,
		workloadClusterID: config.WorkloadClusterID,
	}

	return c, nil
}

func (c *ChartStatusWatcher) Boot(ctx context.Context) {
	go c.watchChartStatus(ctx)
}

// watchChartStatus watches all chart CRs in the target cluster for status
// changes. The matching app CR status is updated otherwise there can be a
// delay of up to 5 minutes until the next resync period.
func (c *ChartStatusWatcher) watchChartStatus(ctx context.Context) {
	for {
		// We need a dynamic client to connect to the cluster. For remote clusters
		// we use the kubeconfig secret but there can be a delay while its
		// created during cluster creation so we wait till it exists.
		dynClient, err := c.waitForDynClient(ctx)
		if err != nil {
			c.logger.Errorf(ctx, err, "failed to get dyn client")
			continue
		}

		// The connection to the cluster will sometimes be down. So we
		// check we can connect and wait with a backoff if it is unavailable.
		err = c.waitForAvailableConnection(ctx, dynClient)
		if err != nil {
			c.logger.Errorf(ctx, err, "failed to get available connection")
			continue
		}

		// We watch all chart CRs to check for status changes.
		res, err := dynClient.Resource(chartResource).Namespace(c.chartNamespace).Watch(ctx, metav1.ListOptions{})
		if err != nil {
			c.logger.Errorf(ctx, err, "failed to watch charts in %#q namespace", c.chartNamespace)
			continue
		}

		c.logger.Debugf(ctx, "watching chart CRs in %#q namespace", c.chartNamespace)

		for r := range res.ResultChan() {
			if r.Type == watch.Bookmark {
				// no-op for unsupported events
				continue
			}

			if r.Type == watch.Error {
				c.logger.Debugf(ctx, "got error event for chart %#q", r.Object)
				continue
			}

			unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r.Object)
			if err != nil {
				c.logger.Errorf(ctx, err, "failed to convert %#v to unstructured object", r.Object)
				continue
			}

			chart := &v1alpha1.Chart{}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, chart)
			if err != nil {
				c.logger.Errorf(ctx, err, "failed to convert unstructured object %#v to chart", unstructuredObj)
				continue
			}

			// The chart CR is always in the giantswarm namespace so the
			// chart CR is annotated with the app CR namespace.
			appNamespace, ok := chart.Annotations[annotation.AppNamespace]
			if !ok {
				c.logger.Debugf(ctx, "failed to get annotation %#q for chart %#q", annotation.AppNamespace, chart.Name)
				continue
			}

			app := v1alpha1.App{}
			err = c.k8sClient.CtrlClient().Get(ctx,
				types.NamespacedName{Name: chart.Name, Namespace: appNamespace},
				&app)
			if err != nil {
				c.logger.Errorf(ctx, err, "failed to get app '%s/%s'", app.Namespace, app.Name)
				continue
			}

			desiredStatus := toAppStatus(*chart)
			currentStatus := key.AppStatus(app)

			if !equals(currentStatus, desiredStatus) {
				if diff := cmp.Diff(currentStatus, desiredStatus); diff != "" {
					c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status for app '%s/%s' has to be updated", app.Namespace, app.Name), "diff", fmt.Sprintf("(-current +desired):\n%s", diff))
				}

				app.Status = desiredStatus
				err = c.k8sClient.CtrlClient().Status().Update(ctx, &app)
				if err != nil {
					c.logger.Errorf(ctx, err, "failed to update status for app '%s/%s'", app.Namespace, app.Name)
					continue
				}

				c.logger.Debugf(ctx, "status set for app '%s/%s'", app.Namespace, app.Name)
			}
		}

		c.logger.Debugf(ctx, "watch channel had been closed, reopening...")
	}
}

// equals assesses the equality of AppStatuses with regards to distinguishing
// fields.
func equals(a, b v1alpha1.AppStatus) bool {
	if a.AppVersion != b.AppVersion {
		return false
	}
	if a.Release.LastDeployed != b.Release.LastDeployed {
		return false
	}
	if a.Release.Reason != b.Release.Reason {
		return false
	}
	if a.Release.Status != b.Release.Status {
		return false
	}
	if a.Version != b.Version {
		return false
	}

	return true
}

// toAppStatus converts the chart CR to an app CR status.
func toAppStatus(chart v1alpha1.Chart) v1alpha1.AppStatus {
	appStatus := v1alpha1.AppStatus{
		AppVersion: chart.Status.AppVersion,
		Release: v1alpha1.AppStatusRelease{
			Reason: chart.Status.Reason,
			Status: chart.Status.Release.Status,
		},
		Version: chart.Status.Version,
	}
	if chart.Status.Release.LastDeployed != nil {
		appStatus.Release.LastDeployed = *chart.Status.Release.LastDeployed
	}

	return appStatus
}
