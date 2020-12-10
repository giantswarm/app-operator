package tcnamespace

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v2/pkg/project"
	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
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

	// Resource is used to bootstrap chart-operator in tenant clusters.
	// So for other apps we can skip this step.
	if key.AppName(cr) != key.ChartOperatorAppName {
		r.logger.Debugf(ctx, "no need to create namespace for %#q", key.AppName(cr))
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if key.InCluster(cr) {
		r.logger.Debugf(ctx, "app %#q in %#q uses InCluster kubeconfig no need to create namespace", cr.Name, cr.Namespace)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if cc.Status.ClusterStatus.IsUnavailable {
		r.logger.Debugf(ctx, "tenant cluster is unavailable")
		r.logger.Debugf(ctx, "canceling resource")
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

	r.logger.Debugf(ctx, "creating namespace %#q in tenant cluster %#q", ns.Name, key.ClusterID(cr))

	ch := make(chan error)

	go func() {
		_, err = cc.Clients.K8s.K8sClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		close(ch)
	}()

	select {
	case <-ch:
		// Fall through.
	case <-time.After(3 * time.Second):
		// Set status so we don't try to connect to the tenant cluster
		// again in this reconciliation loop.
		cc.Status.ClusterStatus.IsUnavailable = true

		r.logger.Debugf(ctx, "timeout creating namespace")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	if apierrors.IsAlreadyExists(err) {
		// fall through
	} else if tenant.IsAPINotAvailable(err) {
		// Set status so we don't try to connect to the tenant cluster
		// again in this reconciliation loop.
		cc.Status.ClusterStatus.IsUnavailable = true

		r.logger.Debugf(ctx, "tenant cluster not available")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "created namespace %#q in tenant cluster %#q", ns.Name, key.ClusterID(cr))

	return nil
}
