package configmap

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/crud"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v7/service/controller/app/controllercontext"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	configMap, err := toConfigMap(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if !isEmpty(configMap) {
		r.logger.Debugf(ctx, "updating configmap %#q in namespace %#q", configMap.Name, configMap.Namespace)

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = cc.Clients.K8s.K8sClient().CoreV1().ConfigMaps(configMap.Namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "updated configmap %#q in namespace %#q", configMap.Name, configMap.Namespace)
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

	delete, err := r.newDeleteChangeForUpdate(ctx, currentConfigMap, desiredConfigMap)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
	patch.SetCreateChange(create)
	patch.SetUpdateChange(update)
	patch.SetDeleteChange(delete)

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

	r.logger.Debugf(ctx, "finding out if the configmap has to be updated")

	updateConfigMap := &corev1.ConfigMap{}
	isModified := !isEmpty(currentConfigMap) && !equals(currentConfigMap, desiredConfigMap)
	if isModified {
		r.logger.Debugf(ctx, "the configmap has to be updated")

		updateConfigMap = desiredConfigMap.DeepCopy()
		updateConfigMap.ResourceVersion = currentConfigMap.ResourceVersion

		return updateConfigMap, nil
	} else {
		r.logger.Debugf(ctx, "the configmap does not have to be updated")
	}

	return updateConfigMap, nil
}
