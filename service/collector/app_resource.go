package collector

import (
	"context"
	"fmt"
	"sync"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	appDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "app"),
		"Managed apps status.",
		[]string{
			labelName,
			labelNamespace,
			labelStatus,
			labelVersion,
		},
		nil,
	)
)

// AppResourceConfig is this collector's configuration struct.
type AppResourceConfig struct {
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	WatchNamespace string
}

// AppResource is the main struct for this collector.
type AppResource struct {
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	watchNamespace string
}

// NewAppResource creates a new AppResource metrics collector
func NewAppResource(config AppResourceConfig) (*AppResource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	c := &AppResource{
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		watchNamespace: config.WatchNamespace,
	}

	return c, nil
}

func (c *AppResource) collectAppStatus(ctx context.Context, ch chan<- prometheus.Metric) {
	r, err := c.g8sClient.ApplicationV1alpha1().Apps(c.watchNamespace).List(metav1.ListOptions{})
	if err != nil {
		c.logger.LogCtx(ctx, "level", "error", "message", "could not get apps", "stack", fmt.Sprintf("%#v", err))
	}

	for _, app := range r.Items {
		ch <- prometheus.MustNewConstMetric(
			appDesc,
			prometheus.GaugeValue,
			gaugeValue,
			app.Name,
			app.Namespace,
			app.Status.Release.Status,
			app.Status.Version,
		)

	}

}

// Collect is the main metrics collection function.
func (c *AppResource) Collect(ch chan<- prometheus.Metric) error {
	ctx := context.Background()

	c.logger.LogCtx(ctx, "level", "debug", "message", "collecting metrics")

	collectFuncs := []func(context.Context, chan<- prometheus.Metric){
		c.collectAppStatus,
	}

	var wg sync.WaitGroup

	for _, collectFunc := range collectFuncs {
		wg.Add(1)

		go func(collectFunc func(ctx context.Context, ch chan<- prometheus.Metric)) {
			defer wg.Done()
			collectFunc(ctx, ch)
		}(collectFunc)
	}

	wg.Wait()

	c.logger.LogCtx(ctx, "level", "debug", "message", "finished collecting metrics")
	return nil
}

// Describe emits the description for the metrics collected here.
func (c *AppResource) Describe(ch chan<- *prometheus.Desc) error {
	ch <- appDesc
	return nil
}
