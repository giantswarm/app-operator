package configmap

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	configMap, err := toConfigMap(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if configMap != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting the %#q configmap", configMap.Name))

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		err = cc.K8sClient.CoreV1().ConfigMaps(configMap.Namespace).Delete(configMap.Name, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted the %#q configmap", configMap.Name))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not delete the %#q configmap", configMap.Name))
	}

	return nil
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	delete, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := controller.NewPatch()
	patch.SetDeleteChange(delete)

	return patch, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentConfigMap, err := toConfigMap(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredConfigMap, err := toConfigMap(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if desiredConfigMap == nil || desiredConfigMap.Name == "" {
		return nil, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding out if the %#q configMap has to be deleted", desiredConfigMap.Name))

	isModified := !isEmpty(currentConfigMap) && equals(currentConfigMap, desiredConfigMap)
	if isModified {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q configMap needs to be deleted", desiredConfigMap.Name))

		return desiredConfigMap, nil
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q configMap does not need to be deleted", desiredConfigMap.Name))
	}

	return nil, nil
}
