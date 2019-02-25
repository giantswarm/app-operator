package index

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	cm, err := toConfigMap(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if cm.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring update of index configmap %#q", cm.Name))

		_, err = r.k8sClient.CoreV1().ConfigMaps(r.indexNamespace).Update(cm)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured update of index configmap %#q", cm.Name))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to update index configmap"))
	}

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentConfigMap, desiredConfigMap interface{}) (*controller.Patch, error) {
	create, err := r.newCreateChange(ctx, currentConfigMap, desiredConfigMap)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, currentConfigMap, desiredConfigMap)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := controller.NewPatch()
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

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if the index configmap has to be updated")

	updateConfigMap := &corev1.ConfigMap{}
	isModified := currentConfigMap != nil && !isEmpty(currentConfigMap) && !equals(currentConfigMap, desiredConfigMap)
	if isModified {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the index configmap has to be updated")
		latest, err := r.k8sClient.CoreV1().ConfigMaps(r.indexNamespace).Get(desiredConfigMap.GetName(), metav1.GetOptions{})
		if err != nil {
			return nil, microerror.Mask(err)
		}

		updateConfigMap = desiredConfigMap.DeepCopy()
		updateConfigMap.ObjectMeta.ResourceVersion = latest.GetResourceVersion()

		return updateConfigMap, nil
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the index configmap does not have to be updated")
	}

	return updateConfigMap, nil
}
