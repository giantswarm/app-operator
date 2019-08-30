package namespace

import (
	"context"
	"fmt"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}
	ns, err := toNamespace(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if ns != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating namespace %#q in tenant cluster %#q", ns.Name, key.ClusterID(cr)))

		_, err = cc.K8sClient.CoreV1().Namespaces().Create(ns)
		if apierrors.IsAlreadyExists(err) {
			// fall through
		} else if tenant.IsAPINotAvailable(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster not available")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			resourcecanceledcontext.SetCanceled(ctx)
			return nil

		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created namespace %#q in tenant cluster %#q", ns.Name, key.ClusterID(cr)))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not create namespace %#q in tenant cluster %#q", namespace, key.ClusterID(cr)))
	}

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentNamespace, err := toNamespace(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredNamespace, err := toNamespace(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var namespaceToCreate *corev1.Namespace
	if currentNamespace == nil {
		namespaceToCreate = desiredNamespace
	}

	return namespaceToCreate, nil
}
