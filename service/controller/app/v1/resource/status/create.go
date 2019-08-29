package status

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	name := cr.GetName()

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if cc.Status.TenantCluster.IsDeleting {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("namespace %#q is being deleted, no need to reconcile resource", cr.Namespace))

		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)

		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding status for chart %#q in namespace %#q", name, r.chartNamespace))

	chart, err := cc.G8sClient.ApplicationV1alpha1().Charts(r.chartNamespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find chart %#q in namespace %#q", name, r.chartNamespace))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

		return nil
	} else if tenant.IsAPINotAvailable(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is not available.")

		// We should not hammer tenant API if it is not available, the tenant cluster
		// might be initializing. We will retry on next reconciliation loop.
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found status for chart %#q in namespace %#q", name, r.chartNamespace))

	chartStatus := key.ChartStatus(*chart)
	desiredStatus := v1alpha1.AppStatus{
		AppVersion: chartStatus.AppVersion,
		Release: v1alpha1.AppStatusRelease{
			LastDeployed: chartStatus.Release.LastDeployed,
			Reason:       chartStatus.Reason,
			Status:       chartStatus.Release.Status,
		},
		Version: chartStatus.Version,
	}

	if !equals(desiredStatus, key.AppStatus(cr)) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting status for app %#q in namespace %#q", name, r.chartNamespace))

		// Get app CR again to ensure the resource version is correct.
		currentCR, err := r.g8sClient.ApplicationV1alpha1().Apps(cr.Namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		currentCR.Status = desiredStatus

		_, err = r.g8sClient.ApplicationV1alpha1().Apps(cr.Namespace).UpdateStatus(currentCR)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status set for app %#q in namespace %#q", name, r.chartNamespace))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status already set for app %#q in namespace %#q", name, r.chartNamespace))
	}

	return nil
}
