package validation

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/app/v5/pkg/validation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v5/pkg/controller/context/reconciliationcanceledcontext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v5/pkg/status"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = r.appValidator.ValidateApp(ctx, cr)
	if validation.IsValidationError(err) {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("validation error %s", err.Error()))

		err = r.updateAppStatus(ctx, cr, err.Error())
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) updateAppStatus(ctx context.Context, cr v1alpha1.App, reason string) error {
	r.logger.Debugf(ctx, "setting status for app %#q in namespace %#q", cr.Name, cr.Namespace)

	// Get app CR again to ensure the resource version is correct.
	currentCR, err := r.g8sClient.ApplicationV1alpha1().Apps(cr.Namespace).Get(ctx, cr.Name, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	currentCR.Status = v1alpha1.AppStatus{
		Release: v1alpha1.AppStatusRelease{
			Reason: reason,
			Status: status.ResourceNotFoundStatus,
		},
	}

	_, err = r.g8sClient.ApplicationV1alpha1().Apps(cr.Namespace).UpdateStatus(ctx, currentCR, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "status set for app %#q in namespace %#q", cr.Name, cr.Namespace)

	return nil
}
