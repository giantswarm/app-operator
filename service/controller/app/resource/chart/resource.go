package chart

import (
	"reflect"
	"strings"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app-operator/v2/pkg/annotation"
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
	G8sClient versioned.Interface
	Logger    micrologger.Logger

	// Settings.
	ChartNamespace string
}

// Resource implements the chart resource.
type Resource struct {
	// Dependencies.
	g8sClient versioned.Interface
	logger    micrologger.Logger

	// Settings.
	chartNamespace string
}

// New creates a new configured chart resource.
func New(config Config) (*Resource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}

	r := &Resource{
		g8sClient: config.G8sClient,
		logger:    config.Logger,

		chartNamespace: config.ChartNamespace,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
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
