package secret

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	secret, err := toSecret(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if secret.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring creation of secret %#q", secret.Name))

		ctlCtx, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = ctlCtx.K8sClient.CoreV1().Secrets(secret.Namespace).Create(secret)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured creation of secret %#q", secret.Name))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to create secret"))
	}
	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, currentResource, desiredResource interface{}) (interface{}, error) {
	currentSecret, err := toSecret(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredSecret, err := toSecret(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding out if the %#q secret has to be created", desiredSecret.Name))

	createSecret := &corev1.Secret{}

	if isEmpty(currentSecret) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q secret needs to be created", desiredSecret.Name))
		createSecret = desiredSecret
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("the %#q secret does not need to be created", desiredSecret.Name))
	}

	return createSecret, nil
}
