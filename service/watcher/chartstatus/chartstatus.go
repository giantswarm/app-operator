package chartstatus

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/giantswarm/app-operator/v4/pkg/annotation"
)

type ChartStatusWatcherConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	ChartNamespace string
	PodNamespace   string
	UniqueApp      bool
}

type ChartStatusWatcher struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	appNamespace   string
	chartNamespace string
	uniqueApp      bool
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
	if config.PodNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.PodNamespace must not be empty", config)
	}

	c := &ChartStatusWatcher{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		// We get a kubeconfig for the cluster from the chart-operator app CR
		// which is in the same namespace as this instance of app-operator.
		appNamespace:   config.PodNamespace,
		chartNamespace: config.ChartNamespace,
		uniqueApp:      config.UniqueApp,
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
		// We need a g8s client to connect to the cluster. For remote clusters
		// we use the kubeconfig secret but there can be a delay while its
		// created during cluster creation so we wait till it exists.
		g8sClient, err := c.waitForG8sClient(ctx)
		if err != nil {
			c.logger.Errorf(ctx, err, "failed to get g8sclient")
			continue
		}

		// The connection to the cluster will sometimes be down. So we
		// check we can connect and wait with a backoff if it is unavailable.
		err = c.waitForAvailableConnection(ctx, g8sClient)
		if err != nil {
			c.logger.Errorf(ctx, err, "failed to get available connection")
			continue
		}

		// We watch all chart CRs to check for status changes.
		res, err := g8sClient.ApplicationV1alpha1().Charts(c.chartNamespace).Watch(ctx, metav1.ListOptions{})
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

			chart, err := key.ToChart(r.Object)
			if err != nil {
				c.logger.Errorf(ctx, err, "failed to convert %#q to chart", r.Object)
				continue
			}

			// The chart CR is always in the giantswarm namespace so the
			// chart CR is annotated with the app CR namespace.
			appNamespace, ok := chart.Annotations[annotation.AppNamespace]
			if !ok {
				c.logger.Debugf(ctx, "failed to get annotation %#q for chart %#q", annotation.AppNamespace, chart.Name)
				continue
			}

			app, err := c.k8sClient.G8sClient().ApplicationV1alpha1().Apps(appNamespace).Get(ctx, chart.Name, metav1.GetOptions{})
			if err != nil {
				c.logger.Errorf(ctx, err, "failed to get app %#q in namespace %#q", app.Namespace, app.Name)
				continue
			}

			desiredStatus := toAppStatus(chart)
			currentStatus := key.AppStatus(*app)

			if !equals(currentStatus, desiredStatus) {
				if diff := cmp.Diff(currentStatus, desiredStatus); diff != "" {
					c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status for app %#q in %#q namespace has to be updated", app.Name, app.Namespace), "diff", fmt.Sprintf("(-current +desired):\n%s", diff))
				}

				app.Status = desiredStatus

				_, err = c.k8sClient.G8sClient().ApplicationV1alpha1().Apps(app.Namespace).UpdateStatus(ctx, app, metav1.UpdateOptions{})
				if err != nil {
					c.logger.Errorf(ctx, err, "failed to update status for app %#q in namespace %#q", app.Name, app.Namespace)
					continue
				}

				c.logger.Debugf(ctx, "status set for app %#q in namespace %#q", app.Name, app.Namespace)
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
