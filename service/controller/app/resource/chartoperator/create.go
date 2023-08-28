package chartoperator

import (
	"context"

	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v8/pkg/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func (r Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	// Deployment name should not have the cluster ID prefix if its present.
	deploymentName := key.AppName(cr)

	// Resource is used to bootstrap chart-operator. So for other apps we can
	// skip this step.
	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.Debugf(ctx, "no need to install chart-operator for %#q", key.AppName(cr))
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "workload cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	{
		r.logger.Debugf(ctx, "finding %#q deployment", deploymentName)

		_, err = cc.Clients.K8s.K8sClient().AppsV1().Deployments(key.Namespace(cr)).Get(ctx, deploymentName, metav1.GetOptions{})
		if err == nil {
			r.logger.Debugf(ctx, "found %#q deployment", deploymentName)
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		} else if apierrors.IsNotFound(err) {
			// no-op
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "did not find %#q deployment", deploymentName)
	}

	// Check whether cluster has a chart-operator helm release yet.
	{
		// Helm release name should not have the cluster ID prefix if its present.
		releaseName := key.AppName(cr)

		r.logger.Debugf(ctx, "finding release %#q", releaseName)

		_, err := cc.Clients.Helm.GetReleaseContent(ctx, key.Namespace(cr), releaseName)
		if tenant.IsAPINotAvailable(err) {
			r.logger.Debugf(ctx, "workload API not available")

			// We should not hammer workload API if it is not available, the workload
			// cluster might be initializing. We will retry on next reconciliation
			// loop.
			r.logger.Debugf(ctx, "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)

			return nil
		} else if helmclient.IsReleaseNotFound(err) {
			r.logger.Debugf(ctx, "did not find release %#q", releaseName)
			r.logger.Debugf(ctx, "installing release %#q", releaseName)

			err = r.installChartOperator(ctx, cr)
			if IsNotReady(err) {
				r.logger.Debugf(ctx, "%#q not ready", releaseName)

				// chart-operator installs the chart CRD in the cluster.
				// So if its not ready we cancel and retry on the next
				// reconciliation loop.
				r.logger.Debugf(ctx, "canceling reconciliation")
				reconciliationcanceledcontext.SetCanceled(ctx)

				return nil
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.Debugf(ctx, "installed release %#q", releaseName)
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "found release %#q", releaseName)

		releaseContent, err := cc.Clients.Helm.GetReleaseContent(ctx, key.Namespace(cr), releaseName)
		if err != nil {
			return microerror.Mask(err)
		}

		switch releaseContent.Status {
		case helmclient.StatusFailed:
			r.logger.Debugf(ctx, "release %#q failed to install", releaseName)
			r.logger.Debugf(ctx, "updating release %#q", releaseName)

			err = r.updateChartOperator(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.Debugf(ctx, "updated release %#q", releaseName)
		case helmclient.StatusPendingInstall, helmclient.StatusUninstalling:
			r.logger.Debugf(ctx, "release %#q stuck in %#s", releaseName, releaseContent.Status)
			r.logger.Debugf(ctx, "delete release %#q", releaseName)

			err = r.uninstallChartOperator(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			err = r.deleteFinalizers(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.Debugf(ctx, "deleted release %#q", releaseName)
		case helmclient.StatusDeployed:
			r.logger.Debugf(ctx, "release %#q deployed", releaseName)
			r.logger.Debugf(ctx, "triggering charts reconciliation")

			// Checks for App CRs without corresponding Chart CRs in the workload cluster,
			// and then annotate them to trigger reconciliation and speed up bootstrapping.
			err = r.triggerReconciliation(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.Debugf(ctx, "triggered charts reconciliation")
		}
	}

	return nil
}
