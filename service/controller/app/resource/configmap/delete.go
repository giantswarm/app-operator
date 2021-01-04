package configmap

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/crud"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v3/service/controller/app/controllercontext"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	configMap, err := toConfigMap(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if !isEmpty(configMap) {
		r.logger.Debugf(ctx, "deleting configmap %#q in namespace %#q", configMap.Name, configMap.Namespace)

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		err = cc.Clients.K8s.K8sClient().CoreV1().ConfigMaps(configMap.Namespace).Delete(ctx, configMap.Name, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "already deleted configmap %#q in namespace %#q", configMap.Name, configMap.Namespace)
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "deleted Chart CR %#q in namespace %#q", configMap.Name, configMap.Namespace)
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
	desiredConfigMap, err := toConfigMap(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return desiredConfigMap, nil
}

func (r *Resource) newDeleteChangeForUpdate(ctx context.Context, currentState, desiredState interface{}) (interface{}, error) {
	currentConfigMap, err := toConfigMap(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	desiredConfigMap, err := toConfigMap(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "finding out if the configmap has to be deleted")

	if !isEmpty(currentConfigMap) && isEmpty(desiredConfigMap) {
		r.logger.Debugf(ctx, "the configmap has to be deleted")
		return currentConfigMap, nil
	}

	r.logger.Debugf(ctx, "the configmap does not have to be deleted")

	return nil, nil
}
