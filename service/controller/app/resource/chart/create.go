package chart

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	chart, err := toChart(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if chart.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = cc.Clients.K8s.G8sClient().ApplicationV1alpha1().Charts(chart.Namespace).Create(ctx, chart, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("already created Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))
	}

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, currentResource, desiredResource interface{}) (interface{}, error) {
	currentChart, err := toChart(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredChart, err := toChart(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding out if the %#q chart has to be created", desiredChart.Name))

	createChart := &v1alpha1.Chart{}

	if isEmpty(currentChart) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q chart needs to be created", desiredChart.Name))
		createChart = desiredChart
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q chart does not need to be created", desiredChart.Name))
	}

	return createChart, nil
}
