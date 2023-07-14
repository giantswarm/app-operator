package status

import (
	"context"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v6/pkg/status"
	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
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

	if r.helmControllerBackend {
		return r.ensureCreatedHelmRelease(ctx, cc, cr)
	}

	return r.ensureCreatedChart(ctx, cc, cr)
}

// ensureCreatedChart takes the status from Chart CR and sets it on the
// App CR.
func (r *Resource) ensureCreatedChart(ctx context.Context, cc *controllercontext.Context, cr v1alpha1.App) error {
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

		chartName := key.ChartName(cr, r.workloadClusterID)

		err := cc.Clients.K8s.CtrlClient().Get(
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

		err := r.ctrlClient.Get(
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

// ensureCreatedHelmRelease takes the status from HelmRelease CR, translates it
// to a desired status, and populates App CR status with it.
func (r *Resource) ensureCreatedHelmRelease(ctx context.Context, cc *controllercontext.Context, cr v1alpha1.App) error {
	var helmRelease helmv2.HelmRelease
	var desiredStatus v1alpha1.AppStatus

	if cc.Status.ChartStatus.Status != "" {
		desiredStatus = v1alpha1.AppStatus{
			Release: v1alpha1.AppStatusRelease{
				Reason: cc.Status.ChartStatus.Reason,
				Status: cc.Status.ChartStatus.Status,
			},
		}
	} else {
		r.logger.Debugf(ctx, "finding status for HelmRelease CR %#q in namespace %#q", cr.Name, cr.Namespace)

		err := cc.Clients.K8s.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace},
			&helmRelease,
		)
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "did not find HelmRelease CR %#q in namespace %#q", cr.Name, cr.Namespace)
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "found status for HelmRelease CR %#q in namespace %#q", cr.Name, cr.Namespace)

		desiredStatus = status.GetDesiredStatus(helmRelease.Status)
	}

	if !equals(desiredStatus, key.AppStatus(cr)) {
		r.logger.Debugf(ctx, "setting status for app %#q in namespace %#q", cr.Name, cr.Namespace)

		// Get app CR again to ensure the resource version is correct.
		var currentCR v1alpha1.App

		err := r.ctrlClient.Get(
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
