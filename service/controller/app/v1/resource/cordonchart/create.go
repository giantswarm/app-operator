package cordonchart

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/annotation"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if cc.Status.TenantCluster.IsDeleting {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("namespace %#q is being deleted, no need to reconcile resource", cr.Namespace))

		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

		return nil
	}

	if key.IsCordoned(cr) {
		err := r.addCordon(ctx, cr, cc.G8sClient)
		if err != nil {
			return microerror.Mask(err)
		}
		return nil
	}

	chart, err := cc.G8sClient.ApplicationV1alpha1().Charts(r.chartNamespace).Get(cr.GetName(), metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	orig := chart.GetAnnotations()
	_, ok1 := orig[annotation.ReplacePrefix(annotation.CordonUntil)]
	_, ok2 := orig[annotation.ReplacePrefix(annotation.CordonReason)]

	if !ok1 || !ok2 {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to patch annotations for chart CR %#q in namespace %#q", cr.Name, r.chartNamespace))
		return nil
	}

	err = r.deleteCordon(ctx, cr, cc.G8sClient)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
