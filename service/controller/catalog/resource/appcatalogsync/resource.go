package appcatalogsync

import (
	"reflect"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	Name = "appcatalogsync"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	UniqueApp bool
}

type Resource struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	uniqueApp bool
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		uniqueApp: config.UniqueApp,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

// equals asseses the equality of AppCatalog with regards to distinguishing
// fields.
func equals(a, b v1alpha1.AppCatalog) bool {
	if a.Name != b.Name {
		return false
	}
	if !reflect.DeepEqual(a.Annotations, b.Annotations) {
		return false
	}
	if !reflect.DeepEqual(a.Labels, b.Labels) {
		return false
	}
	if !reflect.DeepEqual(a.Spec, b.Spec) {
		return false
	}

	return true
}
