package validation

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/app/v6/pkg/validation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v8/pkg/controller/context/reconciliationcanceledcontext"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v6/pkg/status"
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

	var currentCR v1alpha1.App

	// Get app CR again to ensure the resource version is correct.
	err := r.ctrlClient.Get(
		ctx,
		types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace},
		&currentCR,
	)
	if err != nil {
		return microerror.Mask(err)
	}

	currentCR.Status = v1alpha1.AppStatus{
		Release: v1alpha1.AppStatusRelease{
			Reason: reason,
			Status: status.ResourceNotFoundStatus,
		},
	}

	err = r.ctrlClient.Status().Update(ctx, &currentCR)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "status set for app %#q in namespace %#q", cr.Name, cr.Namespace)

	return nil
}
