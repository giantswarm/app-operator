package secret

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v5/pkg/resource/crud"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	secret, err := toSecret(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if !isEmpty(secret) {
		r.logger.Debugf(ctx, "updating secret %#q in namespace %#q", secret.Name, secret.Namespace)

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = cc.Clients.K8s.K8sClient().CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "updated secret %#q in namespace %#q", secret.Name, secret.Namespace)
	}

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentSecret, desiredSecret interface{}) (*crud.Patch, error) {
	create, err := r.newCreateChange(ctx, currentSecret, desiredSecret)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, currentSecret, desiredSecret)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	delete, err := r.newDeleteChangeForUpdate(ctx, currentSecret, desiredSecret)
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
	currentSecret, err := toSecret(currentResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	desiredSecret, err := toSecret(desiredResource)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "finding out if the secret has to be updated")

	updateSecret := &corev1.Secret{}
	isModified := !isEmpty(currentSecret) && !equals(currentSecret, desiredSecret)
	if isModified {
		r.logger.Debugf(ctx, "the secret has to be updated")

		updateSecret = desiredSecret.DeepCopy()
		updateSecret.ObjectMeta.ResourceVersion = currentSecret.ObjectMeta.ResourceVersion

		return updateSecret, nil
	} else {
		r.logger.Debugf(ctx, "the secret does not have to be updated")
	}

	return updateSecret, nil
}
