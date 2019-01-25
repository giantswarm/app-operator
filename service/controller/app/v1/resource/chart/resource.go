package chart

import (
	"context"
	"reflect"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/service/controller/app/v1/kubeconfig"
)

const (
	// Name is the identifier of the resource.
	Name = "chartv1"

	chartAPIVersion           = "application.giantswarm.io"
	chartKind                 = "Chart"
	chartVersionBundleVersion = "0.1.0"
)

// Config represents the configuration used to create a new chart resource.
type Config struct {
	// Dependencies.
	G8sClient      versioned.Interface
	K8sClient      kubernetes.Interface
	KubeConfig     *kubeconfig.KubeConfig
	Logger         micrologger.Logger
	WatchNamespace string
}

// Resource implements the chart resource.
type Resource struct {
	// Dependencies.
	g8sClient      versioned.Interface
	k8sClient      kubernetes.Interface
	kubeConfig     *kubeconfig.KubeConfig
	logger         micrologger.Logger
	watchNamespace string
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	return nil, nil
}

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	return nil
}

// New creates a new configured chart resource.
func New(config Config) (*Resource, error) {
	// Dependencies.
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.KubeConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.KubeConfig must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.WatchNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.WatchNamespace must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		g8sClient:      config.G8sClient,
		kubeConfig:     config.KubeConfig,
		k8sClient:      config.K8sClient,
		logger:         config.Logger,
		watchNamespace: config.WatchNamespace,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

// equals asseses the equality of ReleaseStates with regards to distinguishing fields.
func equals(a, b v1alpha1.Chart) bool {
	if a.Name != b.Name {
		return false
	}
	if !reflect.DeepEqual(a.Spec, b.Spec) {
		return false
	}
	if !reflect.DeepEqual(a.Labels, b.Labels) {
		return false
	}
	if !reflect.DeepEqual(a.Annotations, b.Annotations) {
		return false
	}
	return true
}

// isEmpty checks if a ReleaseState is empty.
func isEmpty(c v1alpha1.Chart) bool {
	return equals(c, v1alpha1.Chart{})
}
