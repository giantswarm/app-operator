package kubeconfigfinalizer

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if key.InCluster(cr) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q do not use kubeconfig secret since it would install the chart in the same cluster", cr.GetName()))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	name := key.KubecConfigSecretName(cr)
	namespace := key.KubecConfigSecretNamespace(cr)

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding kubeconfig secret %#q in namespace %#q", name, namespace))

	kubeConfig, err := r.k8sClient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find kubeconfig secret %#q in namespace %#q", name, namespace))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil

	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found kubeconfig secret %#q in namespace %#q", name, namespace))

	finalizerTag := key.KubeConfigFinalizer(cr)

	if !contains(kubeConfig.Finalizers, finalizerTag) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting finalizer for kubeconfig %#q in namespace %#q", name, namespace))

		kubeConfig.Finalizers = append(kubeConfig.Finalizers, finalizerTag)

		_, err := r.k8sClient.CoreV1().Secrets(namespace).Update(kubeConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finalizer already set for kubeconfig secret %#q in namespace %#q", name, namespace))
	}
	return nil
}

func contains(finalizers []string, matching string) bool {
	for _, f := range finalizers {
		if f == matching {
			return true
		}
	}
	return false
}
