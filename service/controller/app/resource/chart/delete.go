package chart

import (
	"context"

	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v5/pkg/resource/crud"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	cr, err := key.ToApp(obj)
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
		err = r.removeFinalizer(ctx, chart)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if chart != nil && chart.Name != "" {
		r.logger.Debugf(ctx, "deleting Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)

		err = cc.Clients.K8s.CtrlClient().Delete(ctx, chart)
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "already deleted Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "deleted Chart CR %#q in namespace %#q", chart.Name, chart.Namespace)
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
