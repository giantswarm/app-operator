package configmap

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	name := key.ChartConfigMapName(cr)

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if cc.Status.TenantCluster.IsDeleting {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("namespace %#q is being deleted, no need to reconcile resource", cr.Namespace))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	if cc.Status.TenantCluster.IsUnavailable {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is unavailable")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	if key.IsCordoned(cr) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q is cordoned", cr.Name))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding configmap %#q in namespace %#q", name, r.chartNamespace))

	ch := make(chan struct{})

	var configmap *corev1.ConfigMap

	go func() {
		configmap, err = cc.Clients.K8s.K8sClient().CoreV1().ConfigMaps(r.chartNamespace).Get(name, metav1.GetOptions{})
		close(ch)
	}()

	select {
	case <-ch:
		// Fall through.
	case <-time.After(3 * time.Second):
		// Set status so we don't try to connect to the tenant cluster
		// again in this reconciliation loop.
		cc.Status.TenantCluster.IsUnavailable = true

		r.logger.LogCtx(ctx, "level", "debug", "message", "timeout getting configmap")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	if apierrors.IsNotFound(err) {
		// Return early as configmap does not exist.
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find configmap %#q in namespace %#q", name, r.chartNamespace))
		return nil, nil
	} else if tenant.IsAPINotAvailable(err) {
		// Set status so we don't try to connect to the tenant cluster
		// again in this reconciliation loop.
		cc.Status.TenantCluster.IsUnavailable = true

		// We should not hammer tenant API if it is not available. We cancel
		// the reconciliation because its likely following resources will also
		// fail.
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is not available.")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found configmap %#q in namespace %#q", name, r.chartNamespace))

	return configmap, nil
}
