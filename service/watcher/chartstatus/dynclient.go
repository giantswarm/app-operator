package chartstatus

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// waitForDynClient returns a dynamic clientset for watching chart CRs.
// If the target cluster is remote we get this from its kubeconfig secret.
// We use a backoff because there can be a delay while the secret is created.
func (c *ChartStatusWatcher) waitForDynClient(ctx context.Context) (dynamic.Interface, error) {
	var err error

	if c.uniqueApp {
		return c.k8sClient.DynClient(), nil
	}

	var kubeConfigSecret *corev1.Secret
	{
		kubeConfigSecret, err = c.waitForKubeConfig(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var restConfig *rest.Config
	{
		restConfig, err = c.kubeConfig.NewRESTConfigForApp(ctx, kubeConfigSecret.GetName(), kubeConfigSecret.GetNamespace())
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var dynClient dynamic.Interface
	{
		c := rest.CopyConfig(restConfig)

		dynClient, err = dynamic.NewForConfig(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return dynClient, nil
}

// waitForAvailableConnection ensures we can connect to the target cluster if it
// is remote. Sometimes the connection will be unavailable so we list all chart
// CRs to confirm the connection is active.
func (c *ChartStatusWatcher) waitForAvailableConnection(ctx context.Context, dynClient dynamic.Interface) error {
	var err error

	o := func() error {
		// List all chart CRs in the target cluster to confirm the connection
		// is active and the chart CRD is installed.
		_, err = dynClient.Resource(chartResource).Namespace(c.chartNamespace).List(ctx, metav1.ListOptions{})
		if tenant.IsAPINotAvailable(err) {
			c.logger.Debugf(ctx, "workload cluster is not available")
			return microerror.Mask(err)
		} else if IsResourceNotFound(err) {
			c.logger.Debugf(ctx, "chart CRD is not installed")
			return microerror.Mask(err)
		} else if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		c.logger.Debugf(ctx, "failed to get available connection: %#v retrying in %s", err, t)
	}

	// maxWait is 0 since cluster creation may fail.
	b := backoff.NewExponential(0, 30*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// waitForKubeConfig waits until the chart-operator app CR is created and its kubeconfig
// secret exists. We use this to access the remote cluster.
func (c *ChartStatusWatcher) waitForKubeConfig(ctx context.Context) (*corev1.Secret, error) {
	var chartOperatorAppCR v1alpha1.App
	var chartOperatorAppName string
	var kubeConfigSecret corev1.Secret
	var err error

	if c.workloadClusterID != "" {
		chartOperatorAppName = fmt.Sprintf("%s-chart-operator", c.workloadClusterID)
	} else {
		chartOperatorAppName = "chart-operator"
	}

	o := func() error {
		err = c.k8sClient.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: chartOperatorAppName, Namespace: c.podNamespace},
			&chartOperatorAppCR,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		err = c.k8sClient.CtrlClient().Get(
			ctx,
			types.NamespacedName{
				Name:      key.KubeConfigSecretName(chartOperatorAppCR),
				Namespace: key.KubeConfigSecretNamespace(chartOperatorAppCR),
			},
			&kubeConfigSecret,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		c.logger.Debugf(ctx, "failed to get kubeconfig: %#v retrying in %s", err, t)
	}

	// maxWait is 0 since kubeconfig creation may fail.
	b := backoff.NewExponential(0, 30*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return &kubeConfigSecret, nil
}
