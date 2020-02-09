package tcnamespace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/pkg/project"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if key.InCluster(cr) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("app %#q in %#q uses InCluster kubeconfig no need to create namespace", cr.Name, cr.Namespace))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	// Resource is used to bootstrap chart-operator in tenant clusters.
	// So for other apps we can skip this step.
	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no need to create namespace for %#q", key.AppName(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				label.Cluster:      key.ClusterID(cr),
				label.ManagedBy:    project.Name(),
				label.Organization: key.OrganizationID(cr),
			},
		},
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating namespace %#q in tenant cluster %#q", ns.Name, key.ClusterID(cr)))

	ch := make(chan error)

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	go func() {
		_, err = cc.K8sClient.CoreV1().Namespaces().Create(ns)
		ch <- err
	}()

	select {
	case err = <-ch:
		// Fall through.
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			// Set status so we don't try to connect to the tenant cluster
			// again in this reconciliation loop.
			cc.Status.TenantCluster.IsUnavailable = true

			r.logger.LogCtx(ctx, "level", "debug", "message", "timeout creating namespace")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		}
	default:
		// Fall through.
	}

	if apierrors.IsAlreadyExists(err) {
		// fall through
	} else if tenant.IsAPINotAvailable(err) {
		// Set status so we don't try to connect to the tenant cluster
		// again in this reconciliation loop.
		cc.Status.TenantCluster.IsUnavailable = true

		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster not available")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created namespace %#q in tenant cluster %#q", ns.Name, key.ClusterID(cr)))

	return nil
}
