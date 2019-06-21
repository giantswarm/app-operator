package configmap

import (
	"context"
	"fmt"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
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

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding configmap %#q in namespace %#q", name, r.chartNamespace))

	chart, err := cc.K8sClient.CoreV1().ConfigMaps(r.chartNamespace).Get(name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Return early as configmap does not exist.
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find configmap %#q in namespace %#q", name, r.chartNamespace))
		return nil, nil
	} else if apierrors.IsTimeout(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "cluster api timeout.")

		// We should not hammer API if it is not available, the cluster
		// might be initializing. We will retry on next reconciliation loop.
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

		return nil, nil
	} else if !key.InCluster(cr) && tenant.IsAPINotAvailable(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is not available.")

		// We should not hammer tenant API if it is not available, the tenant cluster
		// might be initializing. We will retry on next reconciliation loop.
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found configmap %#q in namespace %#q", name, r.chartNamespace))

	return chart, nil
}
