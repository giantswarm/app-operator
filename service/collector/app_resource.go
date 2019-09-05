package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

var (
	appDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "app", "info"),
		"Managed apps status.",
		[]string{
			labelName,
			labelNamespace,
			labelStatus,
			labelVersion,
		},
		nil,
	)

	appCordonExpireTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "app", "cordon_expire_time_seconds"),
		"A metric of the expire time of cordoned apps unix seconds.",
		[]string{
			labelName,
			labelNamespace,
		},
		nil,
	)
)

// AppResourceConfig is this collector's configuration struct.
type AppResourceConfig struct {
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
}

// AppResource is the main struct for this collector.
type AppResource struct {
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
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
	}

	return c, nil
}

// Collect is the main metrics collection function.
func (c *AppResource) Collect(ch chan<- prometheus.Metric) error {
	ctx := context.Background()

	c.logger.LogCtx(ctx, "level", "debug", "message", "collecting metrics")

	err := c.collectAppStatus(ctx, ch)
	if err != nil {
		return microerror.Mask(err)
	}

	c.logger.LogCtx(ctx, "level", "debug", "message", "finished collecting metrics")
	return nil
}

// Describe emits the description for the metrics collected here.
func (c *AppResource) Describe(ch chan<- *prometheus.Desc) error {
	ch <- appDesc
	ch <- appCordonExpireTimeDesc
	return nil
}

func (c *AppResource) collectAppStatus(ctx context.Context, ch chan<- prometheus.Metric) error {
	r, err := c.g8sClient.ApplicationV1alpha1().Apps("").List(metav1.ListOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	for _, app := range r.Items {
		release := key.AppStatusRelease(app)

		ch <- prometheus.MustNewConstMetric(
			appDesc,
			prometheus.GaugeValue,
			gaugeValue,
			app.Name,
			app.Namespace,
			release.Status,
			app.Status.Version,
		)

		if !key.IsCordoned(app) {
			continue
		}

		t, err := convertToTime(key.CordonUntil(app))
		if err != nil {
			c.logger.Log("level", "warning", "message", "could not convert cordon-until", "stack", fmt.Sprintf("%#v", err))
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			appCordonExpireTimeDesc,
			prometheus.GaugeValue,
			float64(t.Unix()),
			key.AppName(app),
			key.Namespace(app),
		)
	}
	return nil
}

func convertToTime(input string) (time.Time, error) {
	layout := "2006-01-02T15:04:05"

	split := strings.Split(input, ".")
	if len(split) == 0 {
		return time.Time{}, microerror.Maskf(invalidExecutionError, "%#q must have at least one item in order to collect metrics for the cordon expiration", input)
	}

	t, err := time.Parse(layout, split[0])
	if err != nil {
		return time.Time{}, microerror.Maskf(invalidExecutionError, "parsing timestamp %#q failed: %#v", split[0], err.Error())
	}

	return t, nil
}
