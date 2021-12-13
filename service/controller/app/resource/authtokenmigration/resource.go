package authtokenmigration

import (
	"context"

	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v6/pkg/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
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

	// For in-cluster app CRs or if the app is not chart-operator we don't need
	// to take any action.
	if key.InCluster(cr) || key.AppName(cr) != key.ChartOperatorAppName {
		// Just return to avoid making the logs noisy.
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

	// Check if the secret exists.
	_, err = cc.Clients.K8s.K8sClient().CoreV1().Secrets(namespace).Get(ctx, authTokenName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Nothing to do. Just return to avoid making the logs noisy.
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
