package secret

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	configMap, err := toSecret(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if !isEmpty(configMap) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating secret %#q in namespace %#q", configMap.Name, configMap.Namespace))

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = cc.K8sClient.CoreV1().Secrets(configMap.Namespace).Update(configMap)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated secret %#q in namespace %#q", configMap.Name, configMap.Namespace))
	}

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentSecret, desiredSecret interface{}) (*controller.Patch, error) {
	create, err := r.newCreateChange(ctx, currentSecret, desiredSecret)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, currentSecret, desiredSecret)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := controller.NewPatch()
	patch.SetCreateChange(create)
	patch.SetUpdateChange(update)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, currentResource, desiredResource interface{}) (interface{}, error) {
	currentSecret, err := toSecret(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	desiredSecret, err := toSecret(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if the secret has to be updated")

	updateSecret := &corev1.Secret{}
	isModified := !isEmpty(currentSecret) && !equals(currentSecret, desiredSecret)
	if isModified {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the secret has to be updated")

		updateSecret = desiredSecret.DeepCopy()
		updateSecret.ObjectMeta.ResourceVersion = currentSecret.ObjectMeta.ResourceVersion

		return updateSecret, nil
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the secret does not have to be updated")
	}

	return updateSecret, nil
}
