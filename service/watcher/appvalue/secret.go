package appvalue

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

func (c *AppValueWatcher) watchSecret(ctx context.Context) {
	for {
		lo := metav1.ListOptions{
			LabelSelector: label.AppOperatorWatching,
		}

		// Find the highest resourceVersion for each secret.
		secrets, err := c.k8sClient.K8sClient().CoreV1().Secrets(c.secretNamespace).List(ctx, lo)
		if err != nil {
			c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get secrets with label %#q", label.AppOperatorWatching), "stack", fmt.Sprintf("%#v", err))
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

		res, err := c.k8sClient.K8sClient().CoreV1().Secrets(c.secretNamespace).Watch(ctx, lo)
		if err != nil {
			c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get secrets with label %#q", label.AppOperatorWatching), "stack", fmt.Sprintf("%#v", err))
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
					c.logger.Debugf(ctx, "cache missed secret %#q in namespace %#q", secret.Name, secret.Namespace)
					continue
				}

				storedIndex, ok = v.(map[appIndex]bool)
				if !ok {
					c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("expected '%T', got '%T'", map[appIndex]bool{}, v), "stack", fmt.Sprintf("%#v", err))
					continue
				}
			}

			c.logger.Debugf(ctx, "listing apps depends on %#q secret in namespace %#q", secret.Name, secret.Namespace)

			var currentApp v1alpha1.App

			c.appIndexMutex.RLock()
			for app := range storedIndex {
				c.logger.Debugf(ctx, "triggering %#q app update in namespace %#q", app.Name, app.Namespace)

				err = c.k8sClient.CtrlClient().Get(
					ctx,
					types.NamespacedName{Name: app.Name, Namespace: app.Namespace},
					&currentApp,
				)
				if err != nil {
					c.logger.Errorf(ctx, err, "cannot fetch app CR %s/%s", app.Namespace, app.Name)
					continue
				}

				err = c.addAnnotation(ctx, &currentApp, secret.GetResourceVersion(), secretType)
				if err != nil {
					c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to add annotation to app %#q in namespace %#q", app.Name, app.Namespace), "stack", fmt.Sprintf("%#v", err))
					continue
				}

				c.logger.Debugf(ctx, "triggered %#q app update in namespace %#q", app.Name, app.Namespace)

				c.event.Emit(ctx, &currentApp, "AppUpdated", "change to secret %s/%s triggered an update", secret.Namespace, secret.Name)
			}
			c.appIndexMutex.RUnlock()
			c.logger.Debugf(ctx, "listed apps depends on %#q secret in namespace %#q", secret.Name, secret.Namespace)
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
