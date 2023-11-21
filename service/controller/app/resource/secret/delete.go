package secret

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v8/pkg/resource/crud"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	secret, err := toSecret(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if !isEmpty(secret) {
		r.logger.Debugf(ctx, "deleting secret %#q in namespace %#q", secret.Name, secret.Namespace)

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		err = cc.Clients.K8s.K8sClient().CoreV1().Secrets(secret.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "already deleted secret %#q in namespace %#q", secret.Name, secret.Namespace)
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "deleted Chart CR %#q in namespace %#q", secret.Name, secret.Namespace)
		}
	}

	return nil
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	del, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
	patch.SetDeleteChange(del)

	return patch, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	desiredSecret, err := toSecret(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return desiredSecret, nil
}

func (r *Resource) newDeleteChangeForUpdate(ctx context.Context, currentState, desiredState interface{}) (interface{}, error) {
	currentSecret, err := toSecret(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	desiredSecret, err := toSecret(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "finding out if the secret has to be deleted")

	if !isEmpty(currentSecret) && isEmpty(desiredSecret) {
		r.logger.Debugf(ctx, "the secret has to be deleted")
		return currentSecret, nil
	}

	r.logger.Debugf(ctx, "the secret does not have to be deleted")

	return nil, nil
}
