package configmap

import (
	"context"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	configMap, err := toConfigMap(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if !isEmpty(configMap) {
		r.logger.Debugf(ctx, "creating configmap %#q in namespace %#q", configMap.Name, configMap.Namespace)

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		err = cc.Clients.Ctrl.Create(ctx, configMap)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Debugf(ctx, "already created configmap %#q in namespace %#q", configMap.Name, configMap.Namespace)
		} else if tenant.IsAPINotAvailable(err) {
			// We should not hammer tenant API if it is not available, the tenant cluster
			// might be initializing. We will retry on next reconciliation loop.
			r.logger.Debugf(ctx, "tenant cluster is not available.")
			r.logger.Debugf(ctx, "canceling resource")
			resourcecanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "created configmap %#q in namespace %#q", configMap.Name, configMap.Namespace)
		}
	}

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, currentResource, desiredResource interface{}) (interface{}, error) {
	currentConfigMap, err := toConfigMap(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredConfigMap, err := toConfigMap(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "finding out if the configmap has to be created")

	createConfigMap := &corev1.ConfigMap{}

	if isEmpty(currentConfigMap) {
		r.logger.Debugf(ctx, "the configmap needs to be created")
		createConfigMap = desiredConfigMap
	} else {
		r.logger.Debugf(ctx, "the configmap does not need to be created")
	}

	return createConfigMap, nil
}
