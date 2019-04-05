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

	chart, err := toChart(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if chart != nil && chart.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		err = cc.G8sClient.ApplicationV1alpha1().Charts(cr.GetNamespace()).Delete(chart.Name, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("already deleted Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))
		}
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
	currentChart, err := toChart(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredChart, err := toChart(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding out if the %#q chart has to be deleted", desiredChart.Name))

	isModified := !isEmpty(currentChart) && equals(currentChart, desiredChart)
	if isModified {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q chart needs to be deleted", desiredChart.Name))

		return desiredChart, nil
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q chart does not need to be deleted", desiredChart.Name))
	}

	return nil, nil
}
