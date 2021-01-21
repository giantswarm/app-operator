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

	"github.com/giantswarm/app-operator/v3/pkg/annotation"
)

type ChartStatusConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	ChartNamespace string
	PodNamespace   string
	UniqueApp      bool
}

type ChartStatus struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	chartNamespace string
	uniqueApp      bool
	watchNamespace string
}

func NewChartStatus(config ChartStatusConfig) (*ChartStatus, error) {
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

	var watchNamespace string

	if !config.UniqueApp {
		// Only watch app CRs in the current namespace. The label selector
		// excludes the operator's own app CR which has the unique version.
		watchNamespace = config.PodNamespace
	}

	c := &ChartStatus{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		chartNamespace: config.ChartNamespace,
		uniqueApp:      config.UniqueApp,
		watchNamespace: watchNamespace,
	}

	return c, nil
}

func (c *ChartStatus) Boot(ctx context.Context) {
	// Watch for chart status changes.
	go c.watchChartStatus(ctx)
}

func (c *ChartStatus) watchChartStatus(ctx context.Context) {
	for {
		// We need a valid kubeconfig to connect to the cluster as it may be
		// remote. We get this from the chart-operator app CR. The connection
		// to the cluster may sometimes be down so we wait until we can connect.
		g8sClient, err := c.waitForValidKubeConfig(ctx)
		if err != nil {
			c.logger.LogCtx(ctx, "level", "info", "message", "failed to get g8s client", "stack", fmt.Sprintf("%#v", err))
			continue
		}

		// We watch all chart CRs to check for status changes.
		res, err := g8sClient.ApplicationV1alpha1().Charts(c.chartNamespace).Watch(ctx, metav1.ListOptions{})
		if err != nil {
			c.logger.LogCtx(ctx, "level", "info", "message", "failed to watch charts", "stack", fmt.Sprintf("%#v", err))
			continue
		}

		for r := range res.ResultChan() {
			if r.Type == watch.Bookmark {
				// no-op for unsupported events
				continue
			}

			if r.Type == watch.Error {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("got error event: %#q", r.Object))
				continue
			}

			chart, err := key.ToChart(r.Object)
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", "failed to convert chart object", "stack", fmt.Sprintf("%#v", err))
				continue
			}

			// The chart CR is always in the giantswarm namespace so the
			// chart CR is annotated with the app CR namespace.
			appNamespace, ok := chart.Annotations[annotation.AppNamespace]
			if !ok {
				c.logger.LogCtx(ctx, "level", "info", "message", "failed to get app namespace annotation: %#q", r.Object)
				continue
			}

			app, err := c.k8sClient.G8sClient().ApplicationV1alpha1().Apps(appNamespace).Get(ctx, chart.Name, metav1.GetOptions{})
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get app %#q in namespace %#q", app.Name, app.Namespace), "stack", fmt.Sprintf("%#v", err))
				continue
			}

			desiredStatus := toAppStatus(chart)
			currentStatus := key.AppStatus(*app)

			if !equals(currentStatus, desiredStatus) {
				if diff := cmp.Diff(currentStatus, desiredStatus); diff != "" {
					fmt.Printf("app %#q has to be updated, (-current +desired):\n%s", app.Name, diff)
				}

				app.Status = desiredStatus

				_, err = c.k8sClient.G8sClient().ApplicationV1alpha1().Apps(app.Namespace).UpdateStatus(ctx, app, metav1.UpdateOptions{})
				if err != nil {
					c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to update status for app %#q in namespace %#q", app.Name, app.Namespace), "stack", fmt.Sprintf("%#v", err))
					continue
				}
			}
		}

		c.logger.Debugf(ctx, "watch channel had been closed, reopening...")
	}
}

// equals asseses the equality of AppStatuses with regards to distinguishing
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
