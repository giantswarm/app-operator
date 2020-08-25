package chartoperator

import (
	"context"
	"fmt"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient/v2/pkg/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/v2/service/controller/app/key"
)

func (r Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
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
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to install chart-operator for %#q", key.AppName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is unavailable")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding %#q deployment", cr.Name))

		_, err = cc.Clients.K8s.K8sClient().AppsV1().Deployments(key.Namespace(cr)).Get(ctx, cr.Name, metav1.GetOptions{})
		if err == nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %#q deployment", cr.Name))
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		} else if apierrors.IsNotFound(err) {
			// no-op
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find %#q deployment", cr.Name))
	}

	// Check whether cluster has a chart-operator helm release yet.
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding release %#q", cr.Name))

		_, err := cc.Clients.Helm.GetReleaseContent(ctx, key.Namespace(cr), cr.Name)
		if tenant.IsAPINotAvailable(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available")

			// We should not hammer tenant API if it is not available, the tenant
			// cluster might be initializing. We will retry on next reconciliation
			// loop.
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)

			return nil
		} else if helmclient.IsReleaseNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find release %#q", cr.Name))
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing release %#q", cr.Name))

			err = r.installChartOperator(ctx, cr)
			if IsNotReady(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%#q not ready", cr.Name))

				// chart-operator installs the chart CRD in the cluster.
				// So if its not ready we cancel and retry on the next
				// reconciliation loop.
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
				reconciliationcanceledcontext.SetCanceled(ctx)

				return nil
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed release %#q", cr.Name))
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found release %#q", cr.Name))

		releaseContent, err := cc.Clients.Helm.GetReleaseContent(ctx, key.Namespace(cr), cr.Name)
		if err != nil {
			return microerror.Mask(err)
		}

		switch releaseContent.Status {
		case helmclient.StatusFailed, helmclient.StatusPendingInstall:
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("release %#q stuck in %#q", cr.Name, releaseContent.Status))
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating release %#q", cr.Name))

			err = r.updateChartOperator(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated release %#q", cr.Name))
		case helmclient.StatusPendingUpgrade:
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("release %#q stuck in pending-upgrade", cr.Name))
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rollback release %#q", cr.Name))

			err = cc.Clients.Helm.Rollback(ctx, key.Namespace(cr), key.ReleaseName(cr), 0, helmclient.RollbackOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rollbacked release %#q", cr.Name))
		}
	}

	return nil
}
