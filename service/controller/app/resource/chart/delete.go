package chart

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/resource/crud"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
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

		err = cc.Clients.K8s.G8sClient().ApplicationV1alpha1().Charts(chart.Namespace).Delete(chart.Name, &metav1.DeleteOptions{})
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

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	del, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
	patch.SetDeleteChange(del)

	return patch, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	desiredChart, err := toChart(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return desiredChart, nil
}
