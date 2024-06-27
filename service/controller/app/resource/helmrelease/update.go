package helmrelease

import (
	"context"
	"fmt"
	"reflect"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/crud"
	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	helmRelease, err := toHelmRelease(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if helmRelease.Name == "" {
		// no-op
		return nil
	}

	r.logger.Debugf(ctx, "updating HelmRelease CR %#q in namespace %#q", helmRelease.Name, helmRelease.Namespace)

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = cc.Clients.K8s.CtrlClient().Update(ctx, helmRelease)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "updated HelmRelease CR %#q in namespace %#q", helmRelease.Name, helmRelease.Namespace)

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentHelmRelease, desiredHelmRelease interface{}) (*crud.Patch, error) {
	create, err := r.newCreateChange(ctx, currentHelmRelease, desiredHelmRelease)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, currentHelmRelease, desiredHelmRelease)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
	patch.SetCreateChange(create)
	patch.SetUpdateChange(update)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, currentResource, desiredResource interface{}) (interface{}, error) {
	currentHelmRelease, err := toHelmRelease(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// If current release is empty we return empty release from this method, so
	// that the ApplyUpdateChange does nothing, and instead the ApplyCreateChange takes
	// action of creating the resource.
	if reflect.DeepEqual(currentHelmRelease, &helmv2.HelmRelease{}) {
		return &helmv2.HelmRelease{}, nil
	}

	desiredHelmRelease, err := toHelmRelease(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	updateHelmRelease := &helmv2.HelmRelease{}
	resourceVersion := currentHelmRelease.GetResourceVersion()

	// Copy current HelmRelease CR and annotations keeping only the values we need
	// for comparing them.
	currentHelmRelease = copyHelmRelease(currentHelmRelease)
	r.configurePause(currentHelmRelease, desiredHelmRelease)

	if !reflect.DeepEqual(currentHelmRelease, desiredHelmRelease) {
		if diff := cmp.Diff(currentHelmRelease, desiredHelmRelease); diff != "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("HelmRelease %#q has to be updated", currentHelmRelease.Name), "diff", fmt.Sprintf("(-current +desired):\n%s", diff))
		}

		updateHelmRelease = desiredHelmRelease.DeepCopy()
		updateHelmRelease.ObjectMeta.ResourceVersion = resourceVersion

		return updateHelmRelease, nil
	}

	return updateHelmRelease, nil
}
