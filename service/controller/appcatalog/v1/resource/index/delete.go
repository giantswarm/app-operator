package index

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	configMap, err := toConfigMap(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if configMap.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting the %#q configmap", configMap.Name))

		err = r.k8sClient.CoreV1().ConfigMaps(r.indexNamespace).Delete(configMap.Name, &metav1.DeleteOptions{})
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

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding out if the %#q configmap has to be deleted", desiredConfigMap.Name))

	isModified := !isEmpty(currentConfigMap) && equals(currentConfigMap, desiredConfigMap)
	if isModified {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q configmap needs to be deleted", desiredConfigMap.Name))

		return desiredConfigMap, nil
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q configmap does not need to be deleted", desiredConfigMap.Name))
	}

	return nil, nil
}
