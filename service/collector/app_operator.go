package collector

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkglabel "github.com/giantswarm/app-operator/v2/pkg/label"
	"github.com/giantswarm/app-operator/v2/pkg/project"
)

var (
	appOperatorDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "ready", "total"),
		"Gauge with ready app-operator instances per app CR version.",
		[]string{
			labelVersion,
		},
		nil,
	)
)

// AppOperatorConfig is this collector's configuration struct.
type AppOperatorConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

// AppOperator is the main struct for this collector.
type AppOperator struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger
}

// NewAppOperator creates a new AppOperator metrics collector
func NewAppOperator(config AppOperatorConfig) (*AppOperator, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	a := &AppOperator{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return a, nil
}

// Collect is the main metrics collection function.
func (a *AppOperator) Collect(ch chan<- prometheus.Metric) error {
	ctx := context.Background()

	err := a.collectAppOperatorStatus(ctx, ch)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Describe emits the description for the metrics collected here.
func (a *AppOperator) Describe(ch chan<- *prometheus.Desc) error {
	ch <- appOperatorDesc
	return nil
}

func (a *AppOperator) collectAppOperatorStatus(ctx context.Context, ch chan<- prometheus.Metric) error {
	var err error

	appVersions, err := a.collectAppVersions(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	operatorVersions, err := a.collectOperatorVersions(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	for version := range appVersions {
		var ready int32

		if version == project.AppTenantVersion() {
			// There should be a single app-operator instance with major version
			// 1 for Helm 2 tenant clusters.
			ready, err = helm2AppOperatorReady(operatorVersions)
			if err != nil {
				a.logger.LogCtx(ctx, "level", "error", "message", "failed to check helm 2 app-operator ready", "stack", fmt.Sprintf("%#v", err))
				ready = 0
			}
		} else {
			// For all other versions there should be a 1:1 mapping from app CR
			// version to app-operator version.
			result, ok := operatorVersions[version]
			if ok {
				ready = result
			} else {
				a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no app-operator found for version %#q", version))
				ready = 0
			}
		}

		ch <- prometheus.MustNewConstMetric(
			appOperatorDesc,
			prometheus.GaugeValue,
			float64(ready),
			version,
		)
	}

	return nil
}

func (a *AppOperator) collectAppVersions(ctx context.Context) (map[string]bool, error) {
	appVersions := map[string]bool{}

	l, err := a.k8sClient.G8sClient().ApplicationV1alpha1().Apps("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for _, app := range l.Items {
		version := app.Labels[label.AppOperatorVersion]
		appVersions[version] = true
	}

	return appVersions, nil
}

func (a *AppOperator) collectOperatorVersions(ctx context.Context) (map[string]int32, error) {
	operatorVersions := map[string]int32{}

	lo := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", pkglabel.App, project.Name()),
	}
	d, err := a.k8sClient.K8sClient().AppsV1().Deployments("giantswarm").List(ctx, lo)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for _, deploy := range d.Items {
		version := deploy.Labels[pkglabel.AppKubernetesVersion]
		operatorVersions[version] = deploy.Status.ReadyReplicas
	}

	return operatorVersions, nil
}

func helm2AppOperatorReady(operatorVersions map[string]int32) (int32, error) {
	var helm2AppOperators int32

	for version, ready := range operatorVersions {
		v, err := semver.NewVersion(version)
		if err != nil {
			return 0, microerror.Mask(err)
		}

		if v.Major() == 1 {
			helm2AppOperators += ready
		}
	}

	return helm2AppOperators, nil
}
