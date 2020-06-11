package tiller

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/key"
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

	// Resource is used to bootstrap chart-operator in tenant clusters.
	// So for other apps we can skip this step.
	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to install tiller for %#q", key.AppName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if key.InCluster(cr) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q in %#q uses InCluster kubeconfig no need to install tiller", cr.Name, cr.Namespace))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is unavailable")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	_, err = cc.Clients.K8s.AppsV1().Deployments(key.Namespace(cr)).Get(key.AppName(cr), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// no-op
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "chart-operator is taking care of tiller resource already")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	err = r.ensureTillerInstalled(ctx, cc.Clients.Helm)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
