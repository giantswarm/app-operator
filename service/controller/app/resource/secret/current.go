package secret

import (
	"context"

	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appopkey "github.com/giantswarm/app-operator/v6/pkg/key"
	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToApp(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// When the Helm Controller backend is enable, config is located in the same namespace
	// the App CR is located at.
	var name, namespace string
	if r.helmControllerBackend {
		name = appopkey.HelmReleaseSecretName(cr)
		namespace = cr.Namespace
	} else {
		name = key.ChartSecretName(cr)
		namespace = r.chartNamespace
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if cc.Status.ClusterStatus.IsDeleting {
		r.logger.Debugf(ctx, "namespace %#q is being deleted, no need to reconcile resource", cr.Namespace)
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "workload cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	if key.IsAppCordoned(cr) {
		r.logger.Debugf(ctx, "app %#q is cordoned", cr.Name)
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	r.logger.Debugf(ctx, "finding secret %#q in namespace %#q", name, namespace)

	secret, err := cc.Clients.K8s.K8sClient().CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Return early as secret does not exist.
		r.logger.Debugf(ctx, "did not find secret %#q in namespace %#q", name, namespace)
		return nil, nil
	} else if tenant.IsAPINotAvailable(err) {
		// We should not hammer workload API if it is not available, the tenant cluster
		// might be initializing. We will retry on next reconciliation loop.
		r.logger.Debugf(ctx, "workload cluster is not available.")
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "found secret %#q in namespace %#q", name, namespace)

	return secret, nil
}
