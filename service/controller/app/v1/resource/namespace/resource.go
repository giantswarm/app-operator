package namespace

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
)

const (
	// Name is the identifier of the resource.
	Name = "namespacev1"
)

const (
	namespace = "giantswarm"
)

// Config represents the configuration used to create a new namespace resource.
type Config struct {
	Logger micrologger.Logger
}

// Resource implements the namespace resource.
type Resource struct {
	logger micrologger.Logger
}

// New creates a new configured namespace resource.
func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	r := &Resource{
		// Dependencies.
		logger: config.Logger,
	}

	return r, nil
}

// Name returns name of the Resource.
func (r *Resource) Name() string {
	return Name
}

func toNamespace(v interface{}) (*corev1.Namespace, error) {
	if v == nil {
		return nil, nil
	}

	namespace, ok := v.(*corev1.Namespace)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.Namespace{}, v)
	}

	return namespace, nil
}
