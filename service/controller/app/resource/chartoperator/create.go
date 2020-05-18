package chartoperator

import (
	"context"
	"fmt"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/key"
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

	if cc.Status.TenantCluster.IsUnavailable {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster is unavailable")
<<<<<<< HEAD:service/controller/app/resource/chartoperator/create.go
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	// We only bootstrap chart-operator if the app CR uses Helm 3.
	// Helm 2 is managed by the thiccc deployment of app-operator.
	if key.HelmMajorVersion(cr) != "3" {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q not using helm 3", cr.Name))
=======
>>>>>>> master:service/controller/app/v1/resource/chartoperator/create.go
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

<<<<<<< HEAD:service/controller/app/resource/chartoperator/create.go
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding %#q deployment", cr.Name))

		_, err = cc.Clients.K8s.K8sClient().AppsV1().Deployments(key.Namespace(cr)).Get(cr.Name, metav1.GetOptions{})
		if err == nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %#q deployment", cr.Name))
=======
	// Check whether cluster has a chart-operator helm release yet.
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding release %#q", cr.Name))

		_, err := cc.Clients.Helm.GetReleaseContent(ctx, cr.Name)
		if helmclient.IsTillerNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "no healthy tiller pod found")

			// Tiller may not be healthy and we cannot continue without a connection
			// to Tiller. We will retry on next reconciliation loop.
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)

			return nil
		} else if helmclient.IsTillerOutdated(err) {
			// Tiller is upgraded by chart-operator. When we want to upgrade
			// Tiller we deploy a new version of chart-operator. So here we
			// can just cancel the resource.
			r.logger.LogCtx(ctx, "level", "debug", "message", "tiller pod is outdated")
>>>>>>> master:service/controller/app/v1/resource/chartoperator/create.go
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
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found release %#q", cr.Name))

<<<<<<< HEAD:service/controller/app/resource/chartoperator/create.go
			releaseContent, err := cc.Clients.Helm.GetReleaseContent(ctx, key.Namespace(cr), cr.Name)
=======
			releaseContent, err := cc.Clients.Helm.GetReleaseContent(ctx, cr.Name)
>>>>>>> master:service/controller/app/v1/resource/chartoperator/create.go
			if err != nil {
				return microerror.Mask(err)
			}

<<<<<<< HEAD:service/controller/app/resource/chartoperator/create.go
			if releaseContent.Status == helmclient.StatusFailed {
=======
			if releaseContent.Status == "FAILED" {
>>>>>>> master:service/controller/app/v1/resource/chartoperator/create.go
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("release %#q failed to install", cr.Name))
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating release %#q", cr.Name))

				err = r.updateChartOperator(ctx, cr)
				if err != nil {
					return microerror.Mask(err)
				}

				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated release %#q", cr.Name))
			}
		}
	}

	return nil
}
