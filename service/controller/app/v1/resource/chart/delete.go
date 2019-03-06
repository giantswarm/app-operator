package chart

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	chart, err := key.ToChart(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if chart.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting the %#q chart", chart.Name))

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		err = cc.G8sClient.ApplicationV1alpha1().Charts(cr.ObjectMeta.Namespace).Delete(chart.Name, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted the %#q chart", chart.Name))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the chart does not need to be deleted")
	}

	return nil
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	delete, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := controller.NewPatch()
	patch.SetDeleteChange(delete)

	return patch, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentChart, err := key.ToChart(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredChart, err := key.ToChart(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding out if the %#q chart has to be deleted", desiredChart.Name))

	isModified := !isEmpty(currentChart) && equals(currentChart, desiredChart)
	if isModified {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q chart needs to be deleted", desiredChart.Name))

		return &desiredChart, nil
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q chart does not need to be deleted", desiredChart.Name))
	}

	return nil, nil
}
