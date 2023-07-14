package configmap

import (
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/app/v7/pkg/values"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "configmap"

	cordonedStatus = "cordoned"
)

// Config represents the configuration used to create a new configmap resource.
type Config struct {
	// Dependencies.
	Logger micrologger.Logger
	Values *values.Values

	// Settings.
	ChartNamespace        string
	HelmControllerBackend bool
}

// Resource implements the configmap resource.
type Resource struct {
	// Dependencies.
	logger micrologger.Logger
	values *values.Values

	// Settings.
	chartNamespace        string
	helmControllerBackend bool
}

// New creates a new configured configmap resource.
func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Values == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Values must not be empty", config)
	}

	if !config.HelmControllerBackend && config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}

	r := &Resource{
		logger: config.Logger,
		values: config.Values,

		chartNamespace:        config.ChartNamespace,
		helmControllerBackend: config.HelmControllerBackend,
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

// equals asseses the equality of ConfigMaps with regards to distinguishing
// fields.
func equals(a, b *corev1.ConfigMap) bool {
	if a.Name != b.Name {
		return false
	}
	if a.Namespace != b.Namespace {
		return false
	}
	if !reflect.DeepEqual(a.Annotations, b.Annotations) {
		return false
	}

	if len(a.Data) != len(b.Data) {
		return false
	}

	var source, dest map[string]interface{}
	for k := range a.Data {
		source = make(map[string]interface{})
		dest = make(map[string]interface{})

		err := yaml.Unmarshal([]byte(a.Data[k]), &source)
		if err != nil {
			return false
		}

		err = yaml.Unmarshal([]byte(b.Data[k]), &dest)
		if err != nil {
			return false
		}

		if !reflect.DeepEqual(source, dest) {
			return false
		}
	}

	return reflect.DeepEqual(a.Labels, b.Labels)
}

// isEmpty checks if a ConfigMap is empty.
func isEmpty(c *corev1.ConfigMap) bool {
	if c == nil {
		return true
	}

	return equals(c, &corev1.ConfigMap{})
}

// toConfigMap converts the input into a ConfigMap.
func toConfigMap(v interface{}) (*corev1.ConfigMap, error) {
	if v == nil {
		return &corev1.ConfigMap{}, nil
	}

	configMap, ok := v.(*corev1.ConfigMap)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.ConfigMap{}, v)
	}

	return configMap, nil
}
