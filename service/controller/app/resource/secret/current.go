package secret

import (
	"context"

	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v5/pkg/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v4/service/controller/app/controllercontext"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToApp(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	name := key.ChartSecretName(cr)

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

	r.logger.Debugf(ctx, "finding secret %#q in namespace %#q", name, r.chartNamespace)

	secret, err := cc.Clients.K8s.K8sClient().CoreV1().Secrets(r.chartNamespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Return early as secret does not exist.
		r.logger.Debugf(ctx, "did not find secret %#q in namespace %#q", name, r.chartNamespace)
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

	r.logger.Debugf(ctx, "found secret %#q in namespace %#q", name, r.chartNamespace)

	return secret, nil
}
