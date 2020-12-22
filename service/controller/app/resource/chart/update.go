package chart

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/annotation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/crud"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

		_, err = cc.Clients.K8s.G8sClient().ApplicationV1alpha1().Charts(chart.Namespace).Update(ctx, chart, metav1.UpdateOptions{})
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
		idx := map[string]bool{
			"Spec":                   true,
			"ObjectMeta.Labels":      true,
			"ObjectMeta.Annotations": true,
		}

		compareOpt := cmp.FilterPath(func(p cmp.Path) bool {
			return !idx[p.String()]
		}, cmp.Ignore())

		annotationOpt := cmp.FilterPath(func(p cmp.Path) bool {
			return p.String() == "ObjectMeta.Annotations"
		}, cmp.FilterValues(func(current, desired string) bool {
			return !strings.HasPrefix(current, annotation.ChartOperatorPrefix)
		}, cmp.Ignore()))

		if diff := cmp.Diff(currentResource, desiredResource, compareOpt, annotationOpt); diff != "" {
			fmt.Printf("chart %#q have to be updated, (-current +desired):\n%s", currentChart.Name, diff)
		}

		updateChart = desiredChart.DeepCopy()
		updateChart.ObjectMeta.ResourceVersion = currentChart.ObjectMeta.ResourceVersion

		return updateChart, nil
	}

	return updateChart, nil
}
