package chart

import (
	"context"
	"strings"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-operator/v7/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v7/service/internal/indexcache"
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

	// Settings.
	ChartNamespace    string
	WorkloadClusterID string
}

// Resource implements the chart resource.
type Resource struct {
	// Dependencies.
	indexCache indexcache.Interface
	logger     micrologger.Logger

	// Settings.
	chartNamespace    string
	workloadClusterID string
}

// New creates a new configured chart resource.
func New(config Config) (*Resource, error) {
	if config.IndexCache == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.IndexCache$ must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}

	r := &Resource{
		indexCache: config.IndexCache,
		logger:     config.Logger,

		chartNamespace:    config.ChartNamespace,
		workloadClusterID: config.WorkloadClusterID,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
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
func copyAnnotations(current, desired *v1alpha1.Chart) {
	webhookAnnotation := annotation.AppOperatorWebhookURL

	for k, currentValue := range current.Annotations {
		if k == webhookAnnotation {
			// Remove webhook annotation that is no longer used.
			continue
		} else if !strings.HasPrefix(k, annotation.ChartOperatorPrefix) {
			continue
		}

		_, ok := desired.Annotations[k]
		if !ok {
			desired.Annotations[k] = currentValue
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
