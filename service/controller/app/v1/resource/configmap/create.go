package configmap

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	configMap, err := toConfigMap(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if configMap.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring creation of configmap %#q", configMap.Name))

		ctlCtx, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = ctlCtx.K8sClient.CoreV1().ConfigMaps(configMap.Namespace).Create(configMap)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured creation of configmap %#q", configMap.Name))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to create configmap"))
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

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding out if the %#q configmap has to be created", desiredConfigMap.Name))

	createConfigMap := &corev1.ConfigMap{}

	if isEmpty(currentConfigMap) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q configmap needs to be created", desiredConfigMap.Name))
		createConfigMap = desiredConfigMap
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q configmap does not need to be created", desiredConfigMap.Name))
	}

	return createConfigMap, nil
}
