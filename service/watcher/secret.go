package watcher

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	pkglabel "github.com/giantswarm/app-operator/v2/pkg/label"
)

func (c *AppValueWatcher) watchSecret(ctx context.Context) {
	for {
		lo := metav1.ListOptions{
			LabelSelector: pkglabel.Watching,
		}

		// Find the highest resourceVersion for each secret.
		secrets, err := c.k8sClient.K8sClient().CoreV1().Secrets("").List(ctx, lo)
		if err != nil {
			c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get secrets with label %#q", pkglabel.Watching), "stack", fmt.Sprintf("%#v", err))
			continue
		}

		var highestResourceVersion uint64

		for _, secret := range secrets.Items {
			currentResourceVersion, err := getResourceVersion(secret.GetResourceVersion())
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get resourceVersion from secrets %#q in namespace %#q", secret.GetName(), secret.GetNamespace()), "stack", fmt.Sprintf("%#v", err))
				continue
			}
			if highestResourceVersion < currentResourceVersion {
				highestResourceVersion = currentResourceVersion
			}
		}

		c.logger.LogCtx(ctx, "debug", fmt.Sprintf("starting ResourceVersion is %d", highestResourceVersion))

		res, err := c.k8sClient.K8sClient().CoreV1().Secrets("").Watch(ctx, lo)
		if err != nil {
			c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get secrets with label %#q", pkglabel.Watching), "stack", fmt.Sprintf("%#v", err))
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

			secret, err := toSecret(r.Object)
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", "failed to convert secret object", "stack", fmt.Sprintf("%#v", err))
				continue
			}

			v, err := getResourceVersion(secret.GetResourceVersion())
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get resourceVersion from secrets %#q in namespace %#q", secret.GetName(), secret.GetNamespace()), "stack", fmt.Sprintf("%#v", err))
				continue
			}

			if v <= highestResourceVersion {
				// no-op
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("no need to reconcile for the older resourceVersion %d", v))
				continue
			}

			secretIndex := resourceIndex{
				ResourceType: secretType,
				Name:         secret.GetName(),
				Namespace:    secret.GetNamespace(),
			}

			var storedIndex map[appIndex]bool
			{
				v, ok := c.resourcesToApps.Load(secretIndex)
				if !ok {
					c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("cache missed secret %#q in namespace %#q", secret.Name, secret.Namespace))
					continue
				}

				storedIndex, ok = v.(map[appIndex]bool)
				if !ok {
					c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("expected '%T', got '%T'", map[appIndex]bool{}, v), "stack", fmt.Sprintf("%#v", err))
					continue
				}
			}

			c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("listing apps depends on %#q secret in namespace %#q", secret.Name, secret.Namespace))
			for app := range storedIndex {
				c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("triggering %#q app update in namespace %#q", app.Name, app.Namespace))

				err := c.addAnnotation(ctx, app, secret.GetResourceVersion(), secretType)
				if err != nil {
					c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to add annotation to app %#q in namespace %#q", app.Name, app.Namespace), "stack", fmt.Sprintf("%#v", err))
					continue
				}

				c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("triggered %#q app update in namespace %#q", app.Name, app.Namespace))
			}
			c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("listed apps depends on %#q secret in namespace %#q", secret.Name, secret.Namespace))
		}

		c.logger.Log("debug", "watch channel had been closed, reopening...")
	}
}

// toSecret converts the input into a Secret.
func toSecret(v interface{}) (*corev1.Secret, error) {
	if v == nil {
		return &corev1.Secret{}, nil
	}

	secret, ok := v.(*corev1.Secret)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.Secret{}, v)
	}

	return secret, nil
}
