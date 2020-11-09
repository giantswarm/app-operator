package chart

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v3/pkg/annotation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
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
	Logger micrologger.Logger

	// Settings.
	ChartNamespace string
	WebhookBaseURL string
}

// Resource implements the chart resource.
type Resource struct {
	// Dependencies.
	logger micrologger.Logger

	// Settings.
	chartNamespace string
	webhookBaseURL string
}

// New creates a new configured chart resource.
func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}
	if config.WebhookBaseURL == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.WebhookBaseURL must not be empty", config)
	}

	r := &Resource{
		logger: config.Logger,

		chartNamespace: config.ChartNamespace,
		webhookBaseURL: config.WebhookBaseURL,
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

	// `chart-operator` helm release is already deleted by the `chartoperator` resource at this point.
	// So app-operator needs to remove finalizers so the chart-operator chart CR is deleted.
	patch := []patch{
		{
			Op:   "remove",
			Path: "/metadata/finalizers",
		},
	}
	bytes, err := json.Marshal(patch)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting finalizers on Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))

	_, err = cc.Clients.K8s.G8sClient().ApplicationV1alpha1().Charts(chart.Namespace).Patch(ctx, chart.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted finalizers on Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))

	return nil
}

// equals asseses the equality of ReleaseStates with regards to distinguishing fields.
func equals(current, desired *v1alpha1.Chart) bool {
	if current.Name != desired.Name {
		return false
	}
	if !reflect.DeepEqual(current.Spec, desired.Spec) {
		return false
	}
	if !reflect.DeepEqual(current.Labels, desired.Labels) {
		return false
	}

	for k, desiredValue := range desired.Annotations {
		if !strings.HasPrefix(k, annotation.ChartOperatorPrefix) {
			continue
		}

		currentValue, ok := current.Annotations[k]
		if !ok {
			return false
		}
		if currentValue != desiredValue {
			return false
		}
	}

	return true
}

// isEmpty checks if a ReleaseState is empty.
func isEmpty(c *v1alpha1.Chart) bool {
	if c == nil {
		return true
	}

	return equals(c, &v1alpha1.Chart{})
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
