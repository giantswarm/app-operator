package secret

import (
	"context"
	"fmt"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	secret, err := toSecret(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if !isEmpty(secret) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating secret %#q in namespace %#q", secret.Name, secret.Namespace))

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = cc.K8sClient.CoreV1().Secrets(secret.Namespace).Create(secret)
		if apierrors.IsAlreadyExists(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("already created secret %#q in namespace %#q", secret.Name, secret.Namespace))
		} else if tenant.IsAPINotAvailable(err) {
			// We should not hammer tenant API if it is not available, the tenant cluster
			// might be initializing. We will retry on next reconciliation loop.
			r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is not available.")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			resourcecanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created secret %#q in namespace %#q", secret.Name, secret.Namespace))
		}
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

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if the secret has to be created")

	createSecret := &corev1.Secret{}

	if isEmpty(currentSecret) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the secret needs to be created")
		createSecret = desiredSecret
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the secret does not need to be created")
	}

	return createSecret, nil
}
