package chart

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	chart, err := toChart(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if chart.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = cc.G8sClient.ApplicationV1alpha1().Charts(chart.Namespace).Update(chart)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))
	}

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentChart, desiredChart interface{}) (*controller.Patch, error) {
	create, err := r.newCreateChange(ctx, currentChart, desiredChart)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, currentChart, desiredChart)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := controller.NewPatch()
	patch.SetCreateChange(create)
	patch.SetUpdateChange(update)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, currentResource, desiredResource interface{}) (interface{}, error) {
	currentChart, err := toChart(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	desiredChart, err := toChart(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if the chart has to be updated")

	updateChart := &v1alpha1.Chart{}
	isModified := !isEmpty(currentChart) && !equals(currentChart, desiredChart)
	if isModified {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the chart has to be updated")

		updateChart = desiredChart.DeepCopy()
		updateChart.ObjectMeta.ResourceVersion = currentChart.ObjectMeta.ResourceVersion

		return updateChart, nil
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the chart does not have to be updated")
	}

	return updateChart, nil
}
