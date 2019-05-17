package kubeconfigfinalizer

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if key.InCluster(cr) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q do not use kubeconfig secret since it installed the chart in the same cluster", cr.GetName()))
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

	if contains(kubeConfig.Finalizers, finalizerTag) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("clear finalizer for kubeconfig secret %#q in namespace %#q", name, namespace))

		kubeConfig.Finalizers = filter(kubeConfig.Finalizers, finalizerTag)

		_, err := r.k8sClient.CoreV1().Secrets(namespace).Update(kubeConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("kubeconfig secret %#q in namespace %#q do not have matching finalizer", name, namespace))
	}
	return nil
}

func filter(finalizers []string, matching string) []string {
	var ret []string
	for _, f := range finalizers {
		if f != matching {
			ret = append(ret, f)
		}
	}
	return ret
}
