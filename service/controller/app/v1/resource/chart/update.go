package chart

import (
	"context"
	"fmt"
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

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

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	chart, err := key.ToChart(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if chart.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring update of chart %#q", chart.Name))

		g8sClient, err := r.kubeConfig.NewG8sClientForApp(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = g8sClient.ApplicationV1alpha1().Charts(cr.Namespace).Update(&chart)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured update of chart %#q", chart.Name))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to update chart"))
	}
	return nil
}

func (r *Resource) newUpdateChange(ctx context.Context, currentResource, desiredResource interface{}) (interface{}, error) {
	currentChart, err := key.ToChart(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	desiredChart, err := key.ToChart(desiredResource)
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
