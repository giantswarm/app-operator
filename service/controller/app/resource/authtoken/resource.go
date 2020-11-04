package authtoken

import (
	"github.com/ghodss/yaml"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"reflect"
)

const (
	// Name is the identifier of the resource.
	Name = "authSecret"

	authTokenName = "auth-token"
	namespace     = "giantswarm"
)

// Config represents the configuration used to create a new cloud config resource.
type Config struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	WebhookAuthToken string
}

// Resource implements the cloud config resource.
type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	webhookAuthToken string
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

		webhookAuthToken: config.WebhookAuthToken,
	}

	return r, nil
}

// Name returns name of the Resource.
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
