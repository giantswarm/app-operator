package chartoperator

import (
	"context"

	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v6/pkg/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v5/service/controller/app/controllercontext"
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
		r.logger.Debugf(ctx, "finding %#q deployment", cr.Name)

		_, err = cc.Clients.K8s.K8sClient().AppsV1().Deployments(key.Namespace(cr)).Get(ctx, cr.Name, metav1.GetOptions{})
		if err == nil {
			r.logger.Debugf(ctx, "found %#q deployment", cr.Name)
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		} else if apierrors.IsNotFound(err) {
			// no-op
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "did not find %#q deployment", cr.Name)
	}

	// Check whether cluster has a chart-operator helm release yet.
	{
		r.logger.Debugf(ctx, "finding release %#q", cr.Name)

		_, err := cc.Clients.Helm.GetReleaseContent(ctx, key.Namespace(cr), cr.Name)
		if tenant.IsAPINotAvailable(err) {
			r.logger.Debugf(ctx, "workload API not available")

			// We should not hammer workload API if it is not available, the workload
			// cluster might be initializing. We will retry on next reconciliation
			// loop.
			r.logger.Debugf(ctx, "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)

			return nil
		} else if helmclient.IsReleaseNotFound(err) {
			r.logger.Debugf(ctx, "did not find release %#q", cr.Name)
			r.logger.Debugf(ctx, "installing release %#q", cr.Name)

			err = r.installChartOperator(ctx, cr)
			if IsNotReady(err) {
				r.logger.Debugf(ctx, "%#q not ready", cr.Name)

				// chart-operator installs the chart CRD in the cluster.
				// So if its not ready we cancel and retry on the next
				// reconciliation loop.
				r.logger.Debugf(ctx, "canceling reconciliation")
				reconciliationcanceledcontext.SetCanceled(ctx)

				return nil
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.Debugf(ctx, "installed release %#q", cr.Name)
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "found release %#q", cr.Name)

		releaseContent, err := cc.Clients.Helm.GetReleaseContent(ctx, key.Namespace(cr), cr.Name)
		if err != nil {
			return microerror.Mask(err)
		}

		switch releaseContent.Status {
		case helmclient.StatusFailed:
			r.logger.Debugf(ctx, "release %#q failed to install", cr.Name)
			r.logger.Debugf(ctx, "updating release %#q", cr.Name)

			err = r.updateChartOperator(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.Debugf(ctx, "updated release %#q", cr.Name)
		case helmclient.StatusPendingInstall, helmclient.StatusUninstalling:
			r.logger.Debugf(ctx, "release %#q stuck in %#s", cr.Name, releaseContent.Status)
			r.logger.Debugf(ctx, "delete release %#q", cr.Name)

			err = r.uninstallChartOperator(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			err = r.deleteFinalizers(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.Debugf(ctx, "deleted release %#q", cr.Name)
		}
	}

	return nil
}
