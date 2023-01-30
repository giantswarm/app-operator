package chart

import (
	"context"
	"fmt"
	"reflect"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/crud"
	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	chart, err := toChart(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if chart.Name == "" {
		// no-op
		return nil
	}

	r.logger.Debugf(ctx, "updating Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = cc.Clients.K8s.CtrlClient().Update(ctx, chart)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "updated Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)

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

	if reflect.DeepEqual(currentChart, &v1alpha1.Chart{}) {
		return &v1alpha1.Chart{}, nil
	}

	desiredChart, err := toChart(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	updateChart := &v1alpha1.Chart{}

	resourceVersion := currentChart.GetResourceVersion()

	// Copy current chart CR and annotations keeping only the values we need
	// for comparing them.
	currentChart = copyChart(currentChart)
	r.copyAnnotations(currentChart, desiredChart)

	if !reflect.DeepEqual(currentChart, desiredChart) {
		if diff := cmp.Diff(currentChart, desiredChart); diff != "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("chart %#q has to be updated", currentChart.Name), "diff", fmt.Sprintf("(-current +desired):\n%s", diff))
		}

		updateChart = desiredChart.DeepCopy()
		updateChart.ObjectMeta.ResourceVersion = resourceVersion

		return updateChart, nil
	}

	return updateChart, nil
}
