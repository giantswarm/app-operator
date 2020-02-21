package configmap

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/resource/crud"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	configMap, err := toConfigMap(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if !isEmpty(configMap) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating configmap %#q in namespace %#q", configMap.Name, configMap.Namespace))

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		cm, err := cc.K8sClient.K8sClient().CoreV1().ConfigMaps(configMap.Namespace).Update(configMap)
		if err != nil {
			return microerror.Mask(err)
		}

		// Add resource version to the controller context. We set an annotation
		// on the chart CR so changes are applied when the configmap is changed.
		cc.ResourceVersion.ConfigMap = cm.ObjectMeta.ResourceVersion

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated configmap %#q in namespace %#q", configMap.Name, configMap.Namespace))
	}

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentConfigMap, desiredConfigMap interface{}) (*crud.Patch, error) {
	create, err := r.newCreateChange(ctx, currentConfigMap, desiredConfigMap)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, currentConfigMap, desiredConfigMap)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
	patch.SetCreateChange(create)
	patch.SetUpdateChange(update)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, currentResource, desiredResource interface{}) (interface{}, error) {
	currentConfigMap, err := toConfigMap(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	desiredConfigMap, err := toConfigMap(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if the configmap has to be updated")

	updateConfigMap := &corev1.ConfigMap{}
	isModified := !isEmpty(currentConfigMap) && !equals(currentConfigMap, desiredConfigMap)
	if isModified {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the configmap has to be updated")

		updateConfigMap = desiredConfigMap.DeepCopy()
		updateConfigMap.ObjectMeta.ResourceVersion = currentConfigMap.ObjectMeta.ResourceVersion

		return updateConfigMap, nil
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the configmap does not have to be updated")
	}

	return updateConfigMap, nil
}
