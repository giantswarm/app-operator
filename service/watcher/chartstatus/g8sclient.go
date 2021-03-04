package chartstatus

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// waitForG8sClient returns a versioned clientset for watching chart CRs.
// If the target cluster is remote we get this from its kubeconfig secret.
// We use a backoff because there can be a delay while the secret is created.
func (c *ChartStatusWatcher) waitForG8sClient(ctx context.Context) (versioned.Interface, error) {
	var err error

	if c.uniqueApp {
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

	o := func() error {
		kubeConfigName := fmt.Sprintf("%s-kubeconfig", c.appNamespace)
		restConfig, err = kubeConfig.NewRESTConfigForApp(ctx, kubeConfigName, c.appNamespace)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		c.logger.Errorf(ctx, err, "failed to get kubeconfig: retrying in %s", t)
	}

	b := backoff.NewExponential(5*time.Minute, 30*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return nil, microerror.Mask(err)
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

// waitForAvailableG8sClient ensures we can connect to the target cluster if it
// is remote. Sometimes the connection will be unavailable so we list all chart
// CRs to confirm the connection is active.
func (c *ChartStatusWatcher) waitForAvailableG8sClient(ctx context.Context, g8sClient versioned.Interface) error {
	var err error

	o := func() error {
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
			c.logger.Errorf(ctx, err, "failed to get available g8s client: retrying in %s", t)
		}
	}

	b := backoff.NewExponential(5*time.Minute, 30*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
