package helmrelease

import (
	"context"
	"reflect"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	helmRelease, err := toHelmRelease(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	// we got an empty HelmRelease CR, hence skipping
	if helmRelease.Name == "" {
		// no-op
		return nil
	}

	r.logger.Debugf(ctx, "creating HelmRelease CR %#q in namespace %#q", helmRelease.Name, helmRelease.Namespace)

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = cc.Clients.K8s.CtrlClient().Create(ctx, helmRelease)
	if apierrors.IsAlreadyExists(err) {
		r.logger.Debugf(ctx, "already created HelmRelease CR %#q in namespace %#q", helmRelease.Name, helmRelease.Namespace)
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "created HelmRelease CR %#q in namespace %#q", helmRelease.Name, helmRelease.Namespace)

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, currentResource, desiredResource interface{}) (interface{}, error) {
	currentHelmReleae, err := toHelmRelease(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredHelmRelease, err := toHelmRelease(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	createHelmRelease := &helmv2.HelmRelease{}

	// If current release is empty, we need to create it, hence we return
	// desired release from this method.
	if reflect.DeepEqual(currentHelmReleae, &helmv2.HelmRelease{}) {
		r.logger.Debugf(ctx, "the %#q HelmRelease CR needs to be created", desiredHelmRelease.Name)
		createHelmRelease = desiredHelmRelease
	}

	return createHelmRelease, nil
}
