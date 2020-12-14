package secret

import (
	"context"

	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
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
		r.logger.Debugf(ctx, "tenant cluster is unavailable")
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

	var secret corev1.Secret

	err = cc.Clients.Ctrl.Get(ctx,
		types.NamespacedName{Name: name, Namespace: r.chartNamespace},
		&secret)
	if apierrors.IsNotFound(err) {
		// Return early as secret does not exist.
		r.logger.Debugf(ctx, "did not find secret %#q in namespace %#q", name, r.chartNamespace)
		return nil, nil
	} else if tenant.IsAPINotAvailable(err) {
		// We should not hammer tenant API if it is not available, the tenant cluster
		// might be initializing. We will retry on next reconciliation loop.
		r.logger.Debugf(ctx, "tenant cluster is not available.")
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "found secret %#q in namespace %#q", name, r.chartNamespace)

	return secret, nil
}
