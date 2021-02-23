package configmap

import (
	"context"
	"time"

	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v4/service/controller/app/controllercontext"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToApp(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	name := key.ChartConfigMapName(cr)

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

		// Adding cordon status to context
		addStatusToContext(cc, key.CordonReason(cr), cordonedStatus)
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	r.logger.Debugf(ctx, "finding configmap %#q in namespace %#q", name, r.chartNamespace)

	ch := make(chan struct{})

	var configmap *corev1.ConfigMap

	go func() {
		configmap, err = cc.Clients.K8s.K8sClient().CoreV1().ConfigMaps(r.chartNamespace).Get(ctx, name, metav1.GetOptions{})
		close(ch)
	}()

	select {
	case <-ch:
		// Fall through.
	case <-time.After(3 * time.Second):
		// Set status so we don't try to connect to the tenant cluster
		// again in this reconciliation loop.
		cc.Status.ClusterStatus.IsUnavailable = true

		r.logger.Debugf(ctx, "timeout getting configmap")
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	if apierrors.IsNotFound(err) {
		// Return early as configmap does not exist.
		r.logger.Debugf(ctx, "did not find configmap %#q in namespace %#q", name, r.chartNamespace)
		return nil, nil
	} else if tenant.IsAPINotAvailable(err) {
		// Set status so we don't try to connect to the workload cluster
		// again in this reconciliation loop.
		cc.Status.ClusterStatus.IsUnavailable = true

		// We should not hammer tenant API if it is not available. We cancel
		// the reconciliation because its likely following resources will also
		// fail.
		r.logger.Debugf(ctx, "workload cluster is not available.")
		r.logger.Debugf(ctx, "canceling reconciliation")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "found configmap %#q in namespace %#q", name, r.chartNamespace)

	return configmap, nil
}
