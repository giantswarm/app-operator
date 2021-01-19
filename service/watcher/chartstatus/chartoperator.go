package chartstatus

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/key"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/giantswarm/app-operator/v3/pkg/label"
)

func (c *ChartStatus) watchForChartOperatorApp(ctx context.Context) (*v1alpha1.App, error) {
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

			return &app, nil
		}
	}
}
