package appnamespace

import (
	"context"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "appnamespace"
)

// Config represents the configuration used to create a new appnamespace resource.
type Config struct {
	// Dependencies.
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
}

// Resource implements the appnamespace resource.
type Resource struct {
	// Dependencies.
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

// New creates a new configured appnamespace resource.
func New(config Config) (*Resource, error) {
	if config.K8sClient == kubernetes.Interface(nil) {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return r, nil

}

func (*Resource) Name() string {
	return Name
}

// addNamespaceStatusToContext checks whether the namespace app CR belongs to
// is being deleting currently.
func (r *Resource) addNamespaceStatusToContext(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	ns, err := r.k8sClient.CoreV1().Namespaces().Get(ctx, cr.Namespace, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	if ns.GetDeletionTimestamp() != nil {
		cc.Status.ClusterStatus.IsDeleting = true
	} else {
		cc.Status.ClusterStatus.IsDeleting = false
	}

	return nil
}
