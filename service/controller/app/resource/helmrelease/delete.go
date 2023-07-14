package helmrelease

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/crud"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}
	hr, err := toHelmRelease(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if hr != nil && hr.Name != "" {
		r.logger.Debugf(ctx, "deleting HelmRelease CR %#q in namespace %#q", hr.Name, hr.Namespace)

		err = cc.Clients.K8s.CtrlClient().Delete(ctx, hr)
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "already deleted HelmRelease CR %#q in namespace %#q", hr.Name, hr.Namespace)
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "deleted HelmRelease CR %#q in namespace %#q", hr.Name, hr.Namespace)
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
	desiredHelmRelease, err := toHelmRelease(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return desiredHelmRelease, nil
}
