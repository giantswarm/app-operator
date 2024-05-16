package helmreleasestatus

import (
	"context"
	"fmt"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"

	appoplabel "github.com/giantswarm/app-operator/v6/pkg/label"
	"github.com/giantswarm/app-operator/v6/pkg/project"
	"github.com/giantswarm/app-operator/v6/pkg/status"
)

var helmReleaseResource = schema.GroupVersionResource{Group: "helm.toolkit.fluxcd.io", Version: "v2beta1", Resource: "helmreleases"}

type HelmReleaseStatusWatcherConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	PodNamespace      string
	UniqueApp         bool
	WorkloadClusterID string
}

type HelmReleaseStatusWatcher struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	podNamespace      string
	uniqueApp         bool
	workloadClusterID string
}

func NewHelmReleaseStatusWatcher(config HelmReleaseStatusWatcherConfig) (*HelmReleaseStatusWatcher, error) {
	if config.K8sClient == k8sclient.Interface(nil) {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.PodNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.WatchNamespace must not be empty", config)
	}

	if config.PodNamespace == "giantswarm" {
		config.PodNamespace = ""
	}

	c := &HelmReleaseStatusWatcher{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		podNamespace:      config.PodNamespace,
		uniqueApp:         config.UniqueApp,
		workloadClusterID: config.WorkloadClusterID,
	}

	return c, nil
}

func (c *HelmReleaseStatusWatcher) Boot(ctx context.Context) {
	go c.watchHelmReleaseStatus(ctx)
}

func (c *HelmReleaseStatusWatcher) getListOptions(ctx context.Context) (metav1.ListOptions, error) {
	var selector labels.Selector

	if c.workloadClusterID != "" {
		selector = appoplabel.ClusterSelector(c.workloadClusterID)
	} else {
		selector = appoplabel.AppVersionSelector(c.uniqueApp)
	}

	// In addition to the usual selector, we also want to get HelmRelease CRs
	// that are marked as managed by the App Operator. This is to decrease the
	// likelyhood of  reacting on non-App-Platform-related HelmRelease CRs that
	// may carry cluster ID or version.
	managedBy, err := labels.NewRequirement(
		label.ManagedBy,
		selection.Equals,
		[]string{project.Name()},
	)
	if err != nil {
		return metav1.ListOptions{}, microerror.Mask(err)
	}

	selector = selector.Add(*managedBy)

	return metav1.ListOptions{LabelSelector: selector.String()}, nil
}

// watchHelmReleaseStatus watches all HelmRelease CRs for status changes.
// The matching app CR status is updated, otherwise there can be a
// delay of up to 10 minutes until the next resync period.
func (c *HelmReleaseStatusWatcher) watchHelmReleaseStatus(ctx context.Context) {
	for {
		// We need a dynamic client to connect to the cluster.
		dynClient := c.k8sClient.DynClient()

		// The connection to the cluster will sometimes be down. So we
		// check we can connect and wait with a backoff if it is unavailable.
		// NOTE: this does not seem really necessary as we look for HelmRelease
		// CRs in the management cluster. It makes sense in the Chart CRs which we
		// look for in a remote workload cluster, yet it has been left here mostly
		// for consistency between the two resources, especially it does not cause
		// any harm to make this extra check.
		err := c.waitForAvailableConnection(ctx, dynClient)
		if err != nil {
			c.logger.Errorf(ctx, err, "failed to get available connection")
			continue
		}

		c.doWatchStatus(ctx, dynClient)
	}
}

func (c *HelmReleaseStatusWatcher) doWatchStatus(ctx context.Context, client dynamic.Interface) {
	listOptions, err := c.getListOptions(ctx)
	if err != nil {
		c.logger.Error(ctx, err, "failed to get HelmRelease CRs selector")
		return
	}

	// We watch all HelmRelease CRs matching selector to check for status changes.
	// For unique App Operator it means selecting by version from all namespaces.
	// For non-unique App Operators it means selecting by cluster ID from their
	// respective namespaces.
	res, err := client.Resource(helmReleaseResource).Namespace(c.podNamespace).Watch(ctx, listOptions)
	if err != nil {
		c.logger.Errorf(ctx, err, "failed to watch HelmRelease CRs with %#q selector", listOptions.LabelSelector)
		return
	}

	c.logger.Debugf(ctx, "watching HelmRelease CRs with %#q selector", listOptions.LabelSelector)

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

		helmRelease := &helmv2.HelmRelease{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, helmRelease)
		if err != nil {
			c.logger.Errorf(ctx, err, "failed to convert unstructured object %#v to HelmRelease", unstructuredObj)
			continue
		}

		// The HelmRelease CR is named after the App CR and placed in the same namespace,
		// hence its metadata can be used to locate the latter.
		app := v1alpha1.App{}
		err = c.k8sClient.CtrlClient().Get(ctx,
			types.NamespacedName{Name: helmRelease.Name, Namespace: helmRelease.Namespace},
			&app)
		if err != nil {
			c.logger.Errorf(ctx, err, "failed to get app '%s/%s'", app.Namespace, app.Name)
			continue
		}

		// We get desired status the same way we get it in the `status` resource,
		// and then we compare it with the current status.
		desiredStatus := status.GetDesiredStatus(helmRelease.Status)
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

			c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status set for app '%s/%s'", app.Namespace, app.Name))
		}
	}

	c.logger.Debugf(ctx, "watch channel had been closed, reopening...")
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
