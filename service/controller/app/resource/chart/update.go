package chart

import (
	"context"
	"fmt"
	"reflect"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/crud"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v3/service/controller/app/controllercontext"
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

	_, err = cc.Clients.K8s.G8sClient().ApplicationV1alpha1().Charts(chart.Namespace).Update(ctx, chart, metav1.UpdateOptions{})
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
		return &v1alpha1.App{}, nil
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
	copyAnnotations(currentChart, desiredChart)

	if !reflect.DeepEqual(currentChart, desiredChart) {
		if diff := cmp.Diff(currentChart, desiredChart); diff != "" {
			fmt.Printf("chart %#q has to be updated, (-current +desired):\n%s", currentChart.Name, diff)
		}

		updateChart = desiredChart.DeepCopy()
		updateChart.ObjectMeta.ResourceVersion = resourceVersion

		return updateChart, nil
	}

	return updateChart, nil
}
