package chartstatus

import (
	"context"
	"errors"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// waitForCtrlClient returns a controller runtime client for watching chart CRs.
// If the target cluster is remote we get this from its kubeconfig secret.
// We use a backoff because there can be a delay while the secret is created.
func (c *ChartStatusWatcher) waitForCtrlClient(ctx context.Context) (client.Client, error) {
	var err error

	if c.uniqueApp {
		return c.k8sClient.CtrlClient(), nil
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

	var ctrlClient client.Client
	{
		// Extend the global client-go scheme which is used by all the tools under
		// the hood. The scheme is required for the controller-runtime controller to
		// be able to watch for runtime objects of a certain type.
		appSchemeBuilder := runtime.SchemeBuilder(schemeBuilder{
			applicationv1alpha1.AddToScheme,
		})
		err = appSchemeBuilder.AddToScheme(scheme.Scheme)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		mapper, err := apiutil.NewDynamicRESTMapper(rest.CopyConfig(restConfig))
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, microerror.Mask(timeoutError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		ctrlClient, err = client.New(rest.CopyConfig(restConfig), client.Options{Scheme: scheme.Scheme, Mapper: mapper})
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return ctrlClient, nil
}

// waitForAvailableConnection ensures we can connect to the target cluster if it
// is remote. Sometimes the connection will be unavailable so we list all chart
// CRs to confirm the connection is active.
func (c *ChartStatusWatcher) waitForAvailableConnection(ctx context.Context, ctrlClient client.Client) error {
	var err error

	o := func() error {
		// List all chart CRs in the target cluster to confirm the connection
		// is active and the chart CRD is installed.
		chartList := &applicationv1alpha1.ChartList{}
		err := ctrlClient.List(
			ctx,
			chartList,
			client.InNamespace(c.chartNamespace),
		)
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
	var err error

	o := func() error {
		err = c.k8sClient.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: chartOperatorAppName, Namespace: c.appNamespace},
			&chartOperatorAppCR)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		if apierrors.IsNotFound(err) {
			c.logger.Debugf(ctx, "'%s/%s' app CR does not exist yet: retrying in %s", c.appNamespace, chartOperatorAppName, t)
		} else if err != nil {
			c.logger.Errorf(ctx, err, "failed to get '%s/%s' app CR: retrying in %s", c.appNamespace, chartOperatorAppName, t)
		}
	}

	b := backoff.NewExponential(5*time.Minute, 30*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return &chartOperatorAppCR, nil
}

// schemeBuilder is used to extend the known types of the client-go scheme.
type schemeBuilder []func(*runtime.Scheme) error
