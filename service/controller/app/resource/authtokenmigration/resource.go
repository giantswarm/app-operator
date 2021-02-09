package authtokenmigration

import (
	"context"

	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/v3/service/controller/app/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "authtokenmigration"

	authTokenName = "auth-token"
	namespace     = "giantswarm"
)

type Config struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
}

// Resource ensures the auth token secret is deleted as its no longer used.
// It was used to secure the status webhook which was replaced by the chart CR
// status watcher.
type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
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
	}

	return r, nil
}

// EnsureCreated ensures the auth token secret is deleted as its no longer used
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if key.InCluster(cr) {
		r.logger.Debugf(ctx, "app %#q in %#q uses InCluster kubeconfig no need for webhook auth token", cr.Name, cr.Namespace)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.Debugf(ctx, "no need to delete webhook auth token for %#q", key.AppName(cr))
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if key.IsAppCordoned(cr) {
		r.logger.Debugf(ctx, "app %#q is cordoned", cr.Name)
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "workload cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	r.logger.Debugf(ctx, "deleting secret %#q in namespace %#q", authTokenName, namespace)

	err = cc.Clients.K8s.K8sClient().CoreV1().Secrets(namespace).Delete(ctx, authTokenName, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "already deleted secret %#q in namespace %#q", authTokenName, namespace)
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "deleted secret %#q in namespace %#q", authTokenName, namespace)

	return nil
}

func (r Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	// no-op
	return nil
}

// Name returns name of the Resource.
func (r *Resource) Name() string {
	return Name
}
