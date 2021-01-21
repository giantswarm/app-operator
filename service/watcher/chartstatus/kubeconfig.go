package chartstatus

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/app-operator/v3/pkg/label"
)

func (c *ChartStatus) getG8sClient(ctx context.Context) (versioned.Interface, error) {
	app, err := c.watchForChartOperatorApp(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

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

	_, err = g8sClient.ApplicationV1alpha1().Charts(c.watchNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return g8sClient, nil
}

func (c *ChartStatus) watchForChartOperatorApp(ctx context.Context) (v1alpha1.App, error) {
	for {
		lo := metav1.ListOptions{
			LabelSelector: label.ChartOperatorAppSelector(c.uniqueApp),
		}
		res, err := c.k8sClient.G8sClient().ApplicationV1alpha1().Apps(c.watchNamespace).Watch(ctx, lo)
		if err != nil {
			c.logger.LogCtx(ctx, "level", "info", "message", "failed to watch apps", "stack", fmt.Sprintf("%#v", err))
			continue
		}

		for r := range res.ResultChan() {
			if r.Type == watch.Bookmark {
				// no-op for unsupported events
				continue
			}

			if r.Type == watch.Error {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("got error event: %#q", r.Object))
				continue
			}

			app, err := key.ToApp(r.Object)
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", "failed to convert chart object", "stack", fmt.Sprintf("%#v", err))
				continue
			}

			return app, nil
		}
	}
}

func (c *ChartStatus) waitForValidKubeConfig(ctx context.Context) (versioned.Interface, error) {
	// TODO Add backoff.
	return c.getG8sClient(ctx)
}
