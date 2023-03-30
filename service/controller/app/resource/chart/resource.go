package chart

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v6/service/internal/indexcache"
)

const (
	// Name is the identifier of the resource.
	Name = "chart"

	chartAPIVersion            = "application.giantswarm.io"
	chartKind                  = "Chart"
	chartCustomResourceVersion = "1.0.0"
)

// Config represents the configuration used to create a new chart resource.
type Config struct {
	// Dependencies.
	IndexCache indexcache.Interface
	Logger     micrologger.Logger
	CtrlClient client.Client

	// Settings.
	ChartNamespace               string
	WorkloadClusterID            string
	DependencyWaitTimeoutMinutes int
}

// Resource implements the chart resource.
type Resource struct {
	// Dependencies.
	indexCache indexcache.Interface
	logger     micrologger.Logger
	ctrlClient client.Client

	// Settings.
	chartNamespace               string
	workloadClusterID            string
	dependencyWaitTimeoutMinutes int
}

// New creates a new configured chart resource.
func New(config Config) (*Resource, error) {
	if config.IndexCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.IndexCache$ must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.DependencyWaitTimeoutMinutes <= 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.DependencyWaitTimeoutMinutes must be greater than 0", config)
	}

	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}

	r := &Resource{
		indexCache: config.IndexCache,
		logger:     config.Logger,
		ctrlClient: config.CtrlClient,

		chartNamespace:               config.ChartNamespace,
		workloadClusterID:            config.WorkloadClusterID,
		dependencyWaitTimeoutMinutes: config.DependencyWaitTimeoutMinutes,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

// addStatusToContext adds the status to the controller context. It will be
// used to set the CR status in the status resource.
func addStatusToContext(cc *controllercontext.Context, reason, status string) {
	cc.Status = controllercontext.Status{
		ChartStatus: controllercontext.ChartStatus{
			Reason: reason,
			Status: status,
		},
	}
}

func (r *Resource) removeFinalizer(ctx context.Context, chart *v1alpha1.Chart) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(chart.Finalizers) == 0 {
		// Return early as nothing to do.
		return nil
	}

	r.logger.Debugf(ctx, "deleting finalizers on Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)

	modifiedChart := chart.DeepCopy()

	// `chart-operator` helm release is already deleted by the `chartoperator` resource at this point.
	// So app-operator needs to remove finalizers so the chart-operator chart CR is deleted.
	modifiedChart.Finalizers = []string{}

	err = cc.Clients.K8s.CtrlClient().Patch(ctx, modifiedChart, client.MergeFrom(chart))
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "deleted finalizers on Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)

	return nil
}

// copyChart creates a new chart object based on the current chart,
// so later we don't need to show unnecessary differences.
func copyChart(current *v1alpha1.Chart) *v1alpha1.Chart {
	newChart := &v1alpha1.Chart{
		TypeMeta: metav1.TypeMeta{
			Kind:       chartKind,
			APIVersion: chartAPIVersion,
		},
	}

	newChart.Name = current.Name
	newChart.Namespace = current.Namespace

	newChart.Annotations = current.Annotations
	newChart.Labels = current.Labels
	newChart.Spec = *current.Spec.DeepCopy()

	return newChart
}

// copyAnnotations copies annotations from the current to desired chart,
// only if the key has a chart-operator.giantswarm.io prefix.
func (r *Resource) copyAnnotations(current, desired *v1alpha1.Chart) {
	webhookAnnotation := annotation.AppOperatorWebhookURL

	pauseValue := current.Annotations[annotationChartOperatorPause]
	pauseReason := current.Annotations[annotationChartOperatorPauseReason]
	pauseTs := current.Annotations[annotationChartOperatorPauseStarted]

	for k, currentValue := range current.Annotations {
		if k == webhookAnnotation {
			// Remove webhook annotation that is no longer used.
			continue
		} else if k == annotationChartOperatorPause {
			// Pause annotation is specially managed.
			continue
		} else if !strings.HasPrefix(k, annotation.ChartOperatorPrefix) {
			continue
		}

		_, ok := desired.Annotations[k]
		if !ok {
			desired.Annotations[k] = currentValue
		}
	}

	// The pause annotation was not set by app operator but from something else, so we want to keep it.
	if pauseValue != "" && pauseReason == "" {
		desired.Annotations[annotationChartOperatorPause] = pauseValue
	}

	if _, paused := desired.Annotations[annotationChartOperatorPause]; paused {
		// Pause was set by app operator, we want to keep the existing pause timestamp.
		if pauseValue != "" && pauseReason != "" && pauseTs != "" {
			desired.Annotations[annotationChartOperatorPauseStarted] = pauseTs
		}
	}

	// Check if pause timestamp is expired.
	if ts, found := desired.Annotations[annotationChartOperatorPauseStarted]; found {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			// Timestamp invalid, do nothing.
			return
		}

		if time.Since(t) > (time.Minute * time.Duration(r.dependencyWaitTimeoutMinutes)) {
			// Wait timeout is expired, remove pause annotations.
			delete(desired.Annotations, annotationChartOperatorPause)
			delete(desired.Annotations, annotationChartOperatorPauseStarted)
			delete(desired.Annotations, annotationChartOperatorPauseReason)
		}
	}
}

// toChart converts the input into a Chart.
func toChart(v interface{}) (*v1alpha1.Chart, error) {
	if v == nil {
		return &v1alpha1.Chart{}, nil
	}

	chart, ok := v.(*v1alpha1.Chart)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1alpha1.Chart{}, v)
	}

	return chart, nil
}
