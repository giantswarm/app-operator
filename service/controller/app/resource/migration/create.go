package migration

import (
	"context"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

// EnsureCreated makes sure the app is properly migrated from Chart Operator to
// Helm Controller backend. At this point the Chart Operator should be gone from
// the workload cluster, alongside its abstract representation in the management cluster,
// as a consequence of Cluster Apps Operator removing it. Hence we may migrate each
// app by removing its Chart CR with configuration from the workload cluster.
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
		r.logger.Debugf(ctx, "namespace %#q is being deleted, no migrate resources", cr.Namespace)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "workload cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	// As a fail safe we check for Chart Operator, which we need to be gone in order
	// to safely remove the Chart CR without removing an app.
	{
		r.logger.Debugf(ctx, "checking %#q deployment is gone", key.ChartOperatorAppName)

		_, err = cc.MigrationClients.K8s.K8sClient().AppsV1().Deployments(r.chartNamespace).Get(ctx, key.ChartOperatorAppName, metav1.GetOptions{})
		if err == nil {
			r.logger.Debugf(ctx, "found %#q deployment, it must be gone to migrate %#q", key.ChartOperatorAppName, key.AppName(cr))
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		} else if apierrors.IsNotFound(err) {
			// no-op
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "%#q deployment is gone", key.ChartOperatorAppName)
	}

	// The Chart Operator is gone, hence we may remove the Chart CR safely. Since there is nothing to
	// process it we need to remove its finalizer first, otherwise it will get stuck in a deletion state.
	var chart v1alpha1.Chart
	{
		err = cc.MigrationClients.K8s.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: key.ChartName(cr, r.workloadClusterID), Namespace: r.chartNamespace},
			&chart,
		)
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "%#q Chart CR is already gone", key.ChartName(cr, r.workloadClusterID))
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		err = r.removeFinalizer(ctx, &chart)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Deleting Chart CR
	{
		r.logger.Debugf(ctx, "deleting Chart CR %#q in namespace %#q", chart.Name, r.chartNamespace)

		err = cc.MigrationClients.K8s.CtrlClient().Delete(ctx, &chart)
		if apierrors.IsNotFound(err) {
			// no-op
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "deleted Chart CR %#q in namespace %#q", chart.Name, r.chartNamespace)
	}

	// Deleting config
	{
		vals := make([]values, 0)
		if chart.Spec.Config.ConfigMap.Name != "" {
			vals = append(vals, values{
				kind:      "ConfigMap",
				name:      chart.Spec.Config.ConfigMap.Name,
				namespace: chart.Spec.Config.ConfigMap.Namespace,
			})
		}

		if chart.Spec.Config.Secret.Name != "" {
			vals = append(vals, values{
				kind:      "Secret",
				name:      chart.Spec.Config.Secret.Name,
				namespace: chart.Spec.Config.Secret.Namespace,
			})
		}

		for _, v := range vals {
			r.logger.Debugf(ctx, "deleting %#q %#q in namespace %#q", v.kind, v.name, v.namespace)

			if v.kind == "ConfigMap" {
				err = cc.MigrationClients.K8s.K8sClient().
					CoreV1().ConfigMaps(v.namespace).
					Delete(ctx, v.name, metav1.DeleteOptions{})
			} else if v.kind == "Secret" {
				err = cc.MigrationClients.K8s.K8sClient().
					CoreV1().Secrets(v.namespace).
					Delete(ctx, v.name, metav1.DeleteOptions{})
			}
			if apierrors.IsNotFound(err) {
				// no-op
			} else if err != nil {
				return microerror.Mask(err)
			}

			r.logger.Debugf(ctx, "deleted %#q %#q in namespace %#q", v.kind, v.name, v.namespace)
		}
	}

	return nil
}
