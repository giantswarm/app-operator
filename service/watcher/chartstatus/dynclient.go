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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	var chartOperatorAppCR *v1alpha1.App
	{
		chartOperatorAppCR, err = c.waitForChartOperator(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var restConfig *rest.Config
	{
		secretName := key.KubeConfigSecretName(*chartOperatorAppCR)
		secretNamespace := key.KubeConfigSecretNamespace(*chartOperatorAppCR)
		restConfig, err = c.kubeConfig.NewRESTConfigForApp(ctx, secretName, secretNamespace)
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

// waitForChartOperator waits until the app CR is created. We use this app
// CR to get the kubeconfig secret we use to access the remote cluster
func (c *ChartStatusWatcher) waitForChartOperator(ctx context.Context) (*v1alpha1.App, error) {
	var chartOperatorAppCR v1alpha1.App
	var chartOperatorAppName string
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

		return nil
	}

	n := func(err error, t time.Duration) {
		if apierrors.IsNotFound(err) {
			c.logger.Debugf(ctx, "'%s/%s' app CR does not exist yet: retrying in %s", c.podNamespace, chartOperatorAppName, t)
		} else if err != nil {
			c.logger.Errorf(ctx, err, "failed to get '%s/%s' app CR: retrying in %s", c.podNamespace, chartOperatorAppName, t)
		}
	}

	b := backoff.NewExponential(5*time.Minute, 30*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return &chartOperatorAppCR, nil
}
