package secret

import (
	"context"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v6/pkg/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	secret, err := toSecret(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if !isEmpty(secret) {
		r.logger.Debugf(ctx, "creating secret %#q in namespace %#q", secret.Name, secret.Namespace)

		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = cc.Clients.K8s.K8sClient().CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			r.logger.Debugf(ctx, "already created secret %#q in namespace %#q", secret.Name, secret.Namespace)
		} else if tenant.IsAPINotAvailable(err) {
			// We should not hammer workload API if it is not available, the tenant cluster
			// might be initializing. We will retry on next reconciliation loop.
			r.logger.Debugf(ctx, "workload cluster is not available.")
			r.logger.Debugf(ctx, "canceling resource")
			resourcecanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "created secret %#q in namespace %#q", secret.Name, secret.Namespace)
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

	r.logger.Debugf(ctx, "finding out if the secret has to be created")

	createSecret := &corev1.Secret{}

	if isEmpty(currentSecret) {
		r.logger.Debugf(ctx, "the secret needs to be created")
		createSecret = desiredSecret
	} else {
		r.logger.Debugf(ctx, "the secret does not need to be created")
	}

	return createSecret, nil
}
