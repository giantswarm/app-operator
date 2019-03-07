package secret

import (
	"reflect"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// Name is the identifier of the resource.
	Name = "secretv1"
)

// Config represents the configuration used to create a new secret resource.
type Config struct {
	// Dependencies.
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// Settings.
	ProjectName    string
	WatchNamespace string
}

// Resource implements the secret resource.
type Resource struct {
	// Dependencies.
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	// Settings.
	projectName    string
	watchNamespace string
}

// New creates a new configured secret resource.
func New(config Config) (*Resource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ProjectName must not be empty", config)
	}
	if config.WatchNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.WatchNamespace must not be empty", config)
	}

	r := &Resource{
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		projectName:    config.ProjectName,
		watchNamespace: config.WatchNamespace,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
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
	if !reflect.DeepEqual(a.Data, b.Data) {
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
		return nil, nil
	}

	secret, ok := v.(*corev1.Secret)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.Secret{}, v)
	}

	return secret, nil
}
