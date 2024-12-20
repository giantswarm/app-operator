package chart

import (
	"context"
	"reflect"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/app-operator/v7/service/controller/app/controllercontext"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	chart, err := toChart(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if chart.Name == "" {
		// no-op
		return nil
	}

	r.logger.Debugf(ctx, "creating Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = cc.Clients.K8s.CtrlClient().Create(ctx, chart)
	if apierrors.IsAlreadyExists(err) {
		r.logger.Debugf(ctx, "already created Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "created Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)

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

	createChart := &v1alpha1.Chart{}

	if reflect.DeepEqual(currentChart, &v1alpha1.Chart{}) {
		r.logger.Debugf(ctx, "the %#q chart needs to be created", desiredChart.Name)
		createChart = desiredChart
	}

	return createChart, nil
}
