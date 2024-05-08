package helmrelease

import (
	"context"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v6/pkg/status"
	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToApp(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if status.FailedStatus[cc.Status.ChartStatus.Status] {
		r.logger.Debugf(ctx, "app %#q failed to merge configMaps/secrets, no need to reconcile resource", cr.Name)
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	if key.IsAppCordoned(cr) {
		r.logger.Debugf(ctx, "app %#q is cordoned", cr.Name)
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	r.logger.Debugf(ctx, "finding HelmRelease %#q", cr.Name)

	helmRelease := &helmv2.HelmRelease{}
	err = cc.Clients.K8s.CtrlClient().Get(
		ctx,
		types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace},
		helmRelease,
	)
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "did not find HelmRelease %#q in namespace %#q", cr.Name, cr.Namespace)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "found HelmRelease %#q", cr.Name)

	return helmRelease, nil
}
