package validation

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v2/pkg/status"
	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v2/service/controller/app/key"
)

const (
	namespaceNotFoundReason = "namespace is not specified"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if cc.Status.ClusterStatus.IsDeleting {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("namespace %#q is being deleted, no need to reconcile resource", cr.Namespace))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if key.AppConfigMapName(cr) != "" {
		ns := key.AppConfigMapNamespace(cr)
		if ns == "" {
			r.logger.LogCtx(ctx, "level", "warning", "message", "dependent configMaps namespace not found")
			addStatusToContext(cc, namespaceNotFoundReason, status.ResourceNotFoundStatus)

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}

		_, err := r.k8sClient.CoreV1().ConfigMaps(ns).Get(ctx, key.AppConfigMapName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "warning", "message", "dependent configMaps are not found")
			addStatusToContext(cc, err.Error(), status.ResourceNotFoundStatus)

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if key.AppSecretName(cr) != "" {
		ns := key.AppSecretNamespace(cr)
		if ns == "" {
			r.logger.LogCtx(ctx, "level", "warning", "message", "dependent secrets namespace not found")
			addStatusToContext(cc, namespaceNotFoundReason, status.ResourceNotFoundStatus)

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}

		_, err := r.k8sClient.CoreV1().Secrets(ns).Get(ctx, key.AppSecretName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "warning", "message", "dependent secrets are not found")
			addStatusToContext(cc, err.Error(), status.ResourceNotFoundStatus)

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if key.UserConfigMapName(cr) != "" {
		ns := key.UserSecretNamespace(cr)
		if ns == "" {
			r.logger.LogCtx(ctx, "level", "warning", "message", "dependent configmap namespace not found")
			addStatusToContext(cc, namespaceNotFoundReason, status.ResourceNotFoundStatus)

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}

		_, err := r.k8sClient.CoreV1().ConfigMaps(ns).Get(ctx, key.UserConfigMapName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "warning", "message", "dependent configMaps are not found")
			addStatusToContext(cc, err.Error(), status.ResourceNotFoundStatus)

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if key.UserSecretName(cr) != "" {
		ns := key.UserSecretNamespace(cr)
		if ns == "" {
			r.logger.LogCtx(ctx, "level", "warning", "message", "dependent secret namespace not found")
			addStatusToContext(cc, namespaceNotFoundReason, status.ResourceNotFoundStatus)

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}

		_, err := r.k8sClient.CoreV1().Secrets(key.UserSecretNamespace(cr)).Get(ctx, key.UserConfigMapName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "warning", "message", "dependent secrets are not found")
			addStatusToContext(cc, err.Error(), status.ResourceNotFoundStatus)

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if !key.InCluster(cr) {
		_, err := r.k8sClient.CoreV1().Secrets(key.KubecConfigSecretNamespace(cr)).Get(ctx, key.KubecConfigSecretName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "warning", "message", "dependent kubeconfig secrets are not found")
			addStatusToContext(cc, err.Error(), status.ResourceNotFoundStatus)

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
