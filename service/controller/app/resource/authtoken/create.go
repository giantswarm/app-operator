package authtoken

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v2/pkg/project"
	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if key.InCluster(cr) {
		r.logger.Debugf(ctx, "app %#q in %#q uses InCluster kubeconfig no need for webhook auth token", cr.Name, cr.Namespace)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.Debugf(ctx, "no need to install webhook auth token for %#q", key.AppName(cr))
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if key.IsAppCordoned(cr) {
		r.logger.Debugf(ctx, "app %#q is cordoned", cr.Name)
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "tenant cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
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

	r.logger.Debugf(ctx, "finding secret %#q in namespace %#q", authTokenName, namespace)

	var currentSecret corev1.Secret

	err = cc.Clients.Ctrl.Get(ctx,
		types.NamespacedName{Name: authTokenName, Namespace: namespace},
		&currentSecret)
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "creating secret %#q in namespace %#q", authTokenName, namespace)

		err = cc.Clients.Ctrl.Create(ctx, desiredSecret)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "created secret %#q in namespace %#q", authTokenName, namespace)

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "found secret %#q in namespace %#q", authTokenName, namespace)

	if !equals(desiredSecret, &currentSecret) {
		r.logger.Debugf(ctx, "updating secret %#q in namespace %#q", authTokenName, namespace)

		err = cc.Clients.Ctrl.Update(ctx, desiredSecret)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "updated secret %#q in namespace %#q", authTokenName, namespace)
	}

	return nil
}
