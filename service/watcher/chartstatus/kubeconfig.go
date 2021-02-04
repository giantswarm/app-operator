package chartstatus

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/app-operator/v3/pkg/label"
)

// getG8sClient returns a versioned clientset for the kubeconfig used by the
// chart-operator app CR running in the same namespace as the operator.
func (c *ChartStatusWatcher) getG8sClient(ctx context.Context) (versioned.Interface, error) {
	lo := metav1.ListOptions{
		LabelSelector: label.ChartOperatorAppSelector(c.uniqueApp),
	}
	apps, err := c.k8sClient.G8sClient().ApplicationV1alpha1().Apps(c.appNamespace).List(ctx, lo)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(apps.Items) != 1 {
		return nil, microerror.Maskf(executionFailedError, "expected 1 chart-operator app CR got %d", len(apps.Items))
	}

	// We have the chart-operator app CR. Now we get its kubeconfig.
	app := apps.Items[0]

	// App CR uses inCluster so we can reuse the existing client.
	if key.InCluster(app) {
		return c.k8sClient.G8sClient(), nil
	}

	var kubeConfig kubeconfig.Interface
	{
		c := kubeconfig.Config{
			K8sClient: c.k8sClient.K8sClient(),
			Logger:    c.logger,
		}

		kubeConfig, err = kubeconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var restConfig *rest.Config
	{
		restConfig, err = kubeConfig.NewRESTConfigForApp(ctx, key.KubeConfigSecretName(app), key.KubeConfigSecretNamespace(app))
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var g8sClient versioned.Interface
	{
		c := rest.CopyConfig(restConfig)

		g8sClient, err = versioned.NewForConfig(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return g8sClient, nil
}

// waitForActiveKubeConfig gets a kubeconfig for the chart-operator app CR.
// If the target cluster is remote then sometimes the connection will be down
// so we list all chart CRs to confirm the connection is active.
func (c *ChartStatusWatcher) waitForActiveKubeConfig(ctx context.Context) (versioned.Interface, error) {
	var g8sClient versioned.Interface
	var err error

	o := func() error {
		g8sClient, err = c.getG8sClient(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		// List all chart CRs in the target cluster to confirm the connection
		// is active and the chart CRD is installed.
		_, err = g8sClient.ApplicationV1alpha1().Charts(c.chartNamespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		if tenant.IsAPINotAvailable(err) {
			// At times the cluster API may be unavailable so we will retry.
			c.logger.Debugf(ctx, "cluster is not available: retrying in %s", t)
		} else {
			c.logger.Errorf(ctx, err, "failed to get active kubeconfig: retrying in %s", t)
		}
	}

	b := backoff.NewExponential(5*time.Minute, 30*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return g8sClient, nil
}
