package namespace

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "namespacev1"
)

// Config represents the configuration used to create a new namespace resource.
type Config struct {
	// Dependencies.
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// Settings.
	ProjectName string
}

// Resource implements the namespace resource.
type Resource struct {
	// Dependencies.
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

// New creates a new configured namespace resource.
func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
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

func (r *Resource) addNamespaceStatusToContext(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	ns, err := r.k8sClient.CoreV1().Namespaces().Get(cr.Namespace, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	if ns.GetDeletionTimestamp() != nil {
		cc.Status.TenantCluster.IsDeleting = true
	} else {
		cc.Status.TenantCluster.IsDeleting = false
	}

	return nil
}
