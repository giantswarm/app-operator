package secret

import (
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/app/v4/pkg/values"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "secret"
)

// Config represents the configuration used to create a new secret resource.
type Config struct {
	// Dependencies.
	Logger micrologger.Logger
	Values *values.Values

	// Settings.
	ChartNamespace string
}

// Resource implements the secret resource.
type Resource struct {
	// Dependencies.
	logger micrologger.Logger
	values *values.Values

	// Settings.
	chartNamespace string
}

// New creates a new configured secret resource.
func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Values == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Values must not be empty", config)
	}

	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}

	r := &Resource{
		logger: config.Logger,
		values: config.Values,

		chartNamespace: config.ChartNamespace,
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

// equals asseses the equality of Secrets with regards to distinguishing
// fields.
func equals(a, b *corev1.Secret) bool {
	if a.Name != b.Name {
		return false
	}
	if a.Namespace != b.Namespace {
		return false
	}
	if !reflect.DeepEqual(a.Annotations, b.Annotations) {
		return false
	}

	var source, dest map[string]interface{}
	{
		source = make(map[string]interface{})
		dest = make(map[string]interface{})

		err := yaml.Unmarshal(a.Data["values"], &source)
		if err != nil {
			return false
		}

		err = yaml.Unmarshal(b.Data["values"], &dest)
		if err != nil {
			return false
		}
	}

	if !reflect.DeepEqual(source, dest) {
		return false
	}
	if !reflect.DeepEqual(a.Labels, b.Labels) {
		return false
	}

	return true
}

// isEmpty checks if a Secret is empty.
func isEmpty(s *corev1.Secret) bool {
	if s == nil {
		return true
	}

	return equals(s, &corev1.Secret{})
}

// toSecret converts the input into a Secret.
func toSecret(v interface{}) (*corev1.Secret, error) {
	if v == nil {
		return &corev1.Secret{}, nil
	}

	secret, ok := v.(*corev1.Secret)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.Secret{}, v)
	}

	return secret, nil
}
