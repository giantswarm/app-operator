package status

import (
	"context"
	"fmt"
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if cc.Status.TenantCluster.IsDeleting {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("namespace %#q is being deleted, no need to reconcile resource", cr.Namespace))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if cc.Status.TenantCluster.IsUnavailable {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is unavailable")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if key.IsCordoned(cr) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q is cordoned", cr.Name))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding status for chart %#q in namespace %#q", cr.Name, r.chartNamespace))

	chart, err := cc.Clients.G8s.ApplicationV1alpha1().Charts(r.chartNamespace).Get(cr.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find chart %#q in namespace %#q", cr.Name, r.chartNamespace))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	} else if tenant.IsAPINotAvailable(err) {
		// We should not hammer tenant API if it is not available, the tenant cluster
		// might be initializing. We will retry on next reconciliation loop.
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is not available.")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found status for chart %#q in namespace %#q", cr.Name, r.chartNamespace))

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
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting status for app %#q in namespace %#q", cr.Name, cr.Namespace))

		// Get app CR again to ensure the resource version is correct.
		currentCR, err := r.g8sClient.ApplicationV1alpha1().Apps(cr.Namespace).Get(cr.Name, metav1.GetOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		currentCR.Status = desiredStatus

		_, err = r.g8sClient.ApplicationV1alpha1().Apps(cr.Namespace).UpdateStatus(currentCR)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status set for app %#q in namespace %#q", cr.Name, cr.Namespace))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status already set for app %#q in namespace %#q", cr.Name, cr.Namespace))
	}

	return nil
}
