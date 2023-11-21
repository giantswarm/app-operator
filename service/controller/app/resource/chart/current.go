package chart

import (
	"context"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v8/pkg/controller/context/resourcecanceledcontext"
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

	chartName := key.ChartName(cr, r.workloadClusterID)

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if status.FailedStatus[cc.Status.ChartStatus.Status] {
		r.logger.Debugf(ctx, "chart %#q failed to merge configMaps/secrets, no need to reconcile resource", cr.Name)
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	if cc.Status.ClusterStatus.IsDeleting {
		r.logger.Debugf(ctx, "namespace %#q is being deleted, no need to reconcile resource", cr.Namespace)
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "workload cluster is unavailable")
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

	r.logger.Debugf(ctx, "finding chart %#q", chartName)

	chart := &v1alpha1.Chart{}
	err = cc.Clients.K8s.CtrlClient().Get(
		ctx,
		types.NamespacedName{Name: chartName, Namespace: r.chartNamespace},
		chart,
	)
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "did not find chart %#q in namespace %#q", chartName, r.chartNamespace)
		return nil, nil
	} else if tenant.IsAPINotAvailable(err) {
		// We should not hammer workload API if it is not available, the workload cluster
		// might be initializing. We will retry on next reconciliation loop.
		r.logger.Debugf(ctx, "workload cluster is not available.")
		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "found chart %#q", chartName)

	return chart, nil
}
