package chart

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	chart, err := key.ToChart(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if chart.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring creation of chart %#q", chart.Name))

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = cc.G8sClient.ApplicationV1alpha1().Charts(cr.Namespace).Create(&chart)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured creation of chart %#q", chart.Name))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to create chart"))
	}
	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, currentResource, desiredResource interface{}) (interface{}, error) {
	currentChart, err := key.ToChart(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredChart, err := key.ToChart(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding out if the %#q chart has to be created", desiredChart.Name))

	createChart := &v1alpha1.Chart{}

	if isEmpty(currentChart) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q chart needs to be created", desiredChart.Name))
		createChart = &desiredChart
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q chart does not need to be created", desiredChart.Name))
	}

	return createChart, nil
}
