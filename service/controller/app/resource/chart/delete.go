package chart

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/resource/crud"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v2/service/controller/app/key"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}
	chart, err := toChart(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if key.AppName(cr) == key.ChartOperatorAppName {
		// `chart-operator` helm release is already deleted by the `chartoperator` resource at this point.
		// So app-operator needs to remove finalizers so the chart-operator chart CR is deleted.
		patch := []patch{
			{
				Op:   "remove",
				Path: "/metadata/finalizers",
			},
		}

		bytes, err := json.Marshal(patch)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting finalizers on Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))
		_, err = cc.Clients.K8s.G8sClient().ApplicationV1alpha1().Charts(chart.Namespace).Patch(ctx, chart.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted finalizers on Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))
	}

	if chart != nil && chart.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting Chart CR %#q in namespace %#q", chart.Name, chart.Namespace))

		err = cc.Clients.K8s.G8sClient().ApplicationV1alpha1().Charts(chart.Namespace).Delete(ctx, chart.Name, metav1.DeleteOptions{})
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
