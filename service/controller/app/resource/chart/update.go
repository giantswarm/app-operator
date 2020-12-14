package chart

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/crud"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	chart, err := toChart(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if chart.Name != "" {
		r.logger.Debugf(ctx, "updating Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		err = cc.Clients.Ctrl.Update(ctx, chart)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "updated Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)
	}

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentChart, desiredChart interface{}) (*crud.Patch, error) {
	create, err := r.newCreateChange(ctx, currentChart, desiredChart)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, currentChart, desiredChart)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
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

	r.logger.Debugf(ctx, "finding out if the chart has to be updated")

	updateChart := &v1alpha1.Chart{}
	isModified := !isEmpty(currentChart) && !equals(currentChart, desiredChart)
	if isModified {
		r.logger.Debugf(ctx, "the chart has to be updated")

		updateChart = desiredChart.DeepCopy()
		updateChart.ObjectMeta.ResourceVersion = currentChart.ObjectMeta.ResourceVersion

		return updateChart, nil
	} else {
		r.logger.Debugf(ctx, "the chart does not have to be updated")
	}

	return updateChart, nil
}
