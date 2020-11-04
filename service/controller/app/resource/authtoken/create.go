package authtoken

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v2/pkg/annotation"
	"github.com/giantswarm/app-operator/v2/pkg/project"
	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v2/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if key.InCluster(cr) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q in %#q uses InCluster kubeconfig no need for webhook auth token", cr.Name, cr.Namespace))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to install webhook auth token for %#q", key.AppName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if key.IsAppCordoned(cr) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q is cordoned", cr.Name))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is unavailable")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	desiredSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      authTokenName,
			Namespace: namespace,
			Annotations: map[string]string{
				annotation.Notes: fmt.Sprintf("DO NOT EDIT. Values managed by %s.", project.Name()),
			},
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Data: map[string][]byte{
			"token": []byte(r.webhookAuthToken),
		},
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding secret %#q in namespace %#q", authTokenName, namespace))

	currentSecret, err := cc.Clients.K8s.K8sClient().CoreV1().Secrets(namespace).Get(ctx, authTokenName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating secret %#q in namespace %#q", authTokenName, namespace))

		_, err := cc.Clients.K8s.K8sClient().CoreV1().Secrets(namespace).Create(ctx, desiredSecret, metav1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created secret %#q in namespace %#q", authTokenName, namespace))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found secret %#q in namespace %#q", authTokenName, namespace))

	if !equals(desiredSecret, currentSecret) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating secret %#q in namespace %#q", authTokenName, namespace))

		_, err := cc.Clients.K8s.K8sClient().CoreV1().Secrets(namespace).Update(ctx, desiredSecret, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated secret %#q in namespace %#q", authTokenName, namespace))
	}

	return nil
}
