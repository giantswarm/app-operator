package chartoperator

import (
	"context"

	"github.com/giantswarm/app/v8/pkg/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/reconciliationcanceledcontext"
	"k8s.io/apimachinery/pkg/types"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/giantswarm/app-operator/v7/service/controller/app/controllercontext"
)

func (r Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	// Resource is used to bootstrap chart-operator. So for other apps we can
	// skip this step.
	if key.AppName(cr) != key.ChartOperatorAppName {
		return nil
	}

	// Check if cluster is being deleted
	clusterID := key.ClusterID(cr)

	if clusterID != "" {
		capiCluster := &capi.Cluster{}
		err = r.ctrlClient.Get(
			ctx,
			types.NamespacedName{Name: clusterID, Namespace: cr.Namespace},
			capiCluster,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		if capiCluster.GetDeletionTimestamp() != nil {
			// Canceling reconciliation stops processing of further
			// resource and makes the OperatorKit to remove the finalizer.
			// This is what we want for Chart Operator upon cluster deletion.
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}
	}

	if cc.Status.ClusterStatus.IsDeleting {
		r.logger.Debugf(ctx, "namespace %#q is being deleted, no need to reconcile resource", cr.Namespace)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "workload cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	err = r.uninstallChartOperator(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
