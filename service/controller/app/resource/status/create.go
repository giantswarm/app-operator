package status

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if cc.Status.ClusterStatus.IsDeleting {
		r.logger.Debugf(ctx, "namespace %#q is being deleted, no need to reconcile resource", cr.Namespace)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	var chart v1alpha1.Chart
	var desiredStatus v1alpha1.AppStatus

	if cc.Status.ChartStatus.Status != "" {
		desiredStatus = v1alpha1.AppStatus{
			Release: v1alpha1.AppStatusRelease{
				Reason: cc.Status.ChartStatus.Reason,
				Status: cc.Status.ChartStatus.Status,
			},
		}
	} else {
		if cc.Status.ClusterStatus.IsUnavailable {
			r.logger.Debugf(ctx, "workload cluster is unavailable")
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		}

		r.logger.Debugf(ctx, "finding status for chart %#q in namespace %#q", cr.Name, r.chartNamespace)

		chartName := formatChartName(cr, r.workloadClusterID)

		err = cc.Clients.K8s.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: chartName, Namespace: r.chartNamespace},
			&chart,
		)
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "did not find chart %#q in namespace %#q", cr.Name, r.chartNamespace)
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		} else if tenant.IsAPINotAvailable(err) {
			// We should not hammer tenant API if it is not available, the workload cluster
			// might be initializing. We will retry on next reconciliation loop.
			r.logger.Debugf(ctx, "workload cluster is not available.")
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "found status for chart %#q in namespace %#q", cr.Name, r.chartNamespace)

		chartStatus := key.ChartStatus(chart)
		desiredStatus = v1alpha1.AppStatus{
			AppVersion: chartStatus.AppVersion,
			Release: v1alpha1.AppStatusRelease{
				Reason: chartStatus.Reason,
				Status: chartStatus.Release.Status,
			},
			Version: chartStatus.Version,
		}
		if chartStatus.Release.LastDeployed != nil {
			desiredStatus.Release.LastDeployed = *chartStatus.Release.LastDeployed
		}
	}

	if !equals(desiredStatus, key.AppStatus(cr)) {
		r.logger.Debugf(ctx, "setting status for app %#q in namespace %#q", cr.Name, cr.Namespace)

		// Get app CR again to ensure the resource version is correct.
		var currentCR v1alpha1.App

		err = r.ctrlClient.Get(
			ctx,
			types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace},
			&currentCR,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		currentCR.Status = desiredStatus

		err = r.ctrlClient.Status().Update(ctx, &currentCR)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "status set for app %#q in namespace %#q", cr.Name, cr.Namespace)
	} else {
		r.logger.Debugf(ctx, "status already set for app %#q in namespace %#q", cr.Name, cr.Namespace)
	}

	return nil
}

func formatChartName(app v1alpha1.App, clusterID string) string {
	// Chart CR name should match the app CR name when installed in the
	// same cluster.
	if key.InCluster(app) {
		return app.Name
	}

	// If the app CR has the cluster ID as a prefix or suffix we remove it
	// as its redundant in the remote cluster.
	chartName := strings.TrimPrefix(app.Name, fmt.Sprintf("%s-", clusterID))
	return strings.TrimSuffix(chartName, fmt.Sprintf("-%s", clusterID))
}
