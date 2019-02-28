package configmap

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	K8sClient   kubernetes.Interface
	Logger      micrologger.Logger
	StateGetter StateGetter

	Name string
}

type Resource struct {
	k8sClient   kubernetes.Interface
	logger      micrologger.Logger
	stateGetter StateGetter

	name string
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.StateGetter == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.StateGetter must not be empty", config)
	}

	if config.Name == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Name must not be empty", config)
	}

	r := &Resource{
		k8sClient:   config.K8sClient,
		logger:      config.Logger,
		stateGetter: config.StateGetter,

		name: config.Name,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return r.name
}

func containsConfigMap(cr *v1.ConfigMap, crs []*v1.ConfigMap) bool {
	for _, a := range crs {
		if cr.Name == a.Name && cr.Namespace == a.Namespace {
			return true
		}
	}

	return false
}

func toConfigMaps(v interface{}) ([]*v1.ConfigMap, error) {
	x, ok := v.([]*v1.ConfigMap)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", x, v)
	}

	return x, nil
}
