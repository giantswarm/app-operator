package appvalue

import (
	"context"
	"fmt"
	"strconv"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *AppValueWatcher) watchConfigMap(ctx context.Context) {
	for {
		lo := metav1.ListOptions{
			LabelSelector: label.AppOperatorWatching,
		}

		// Find the highest resourceVersion for each configmap.
		cms, err := c.k8sClient.K8sClient().CoreV1().ConfigMaps("").List(ctx, lo)
		if err != nil {
			c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get configmaps with label %#q", label.AppOperatorWatching), "stack", fmt.Sprintf("%#v", err))
			continue
		}

		var highestResourceVersion uint64

		for _, cm := range cms.Items {
			currentResourceVersion, err := getResourceVersion(cm.GetResourceVersion())
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get resourceVersion from configmaps %#q in namespace %#q", cm.GetName(), cm.GetNamespace()), "stack", fmt.Sprintf("%#v", err))
				continue
			}
			if highestResourceVersion < currentResourceVersion {
				highestResourceVersion = currentResourceVersion
			}
		}

		c.logger.LogCtx(ctx, "debug", fmt.Sprintf("starting ResourceVersion is %d", highestResourceVersion))

		res, err := c.k8sClient.K8sClient().CoreV1().ConfigMaps("").Watch(ctx, lo)
		if err != nil {
			c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get configmaps with label %#q", label.AppOperatorWatching), "stack", fmt.Sprintf("%#v", err))
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

			cm, err := toConfigMap(r.Object)
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", "failed to convert configmap object", "stack", fmt.Sprintf("%#v", err))
				continue
			}

			v, err := getResourceVersion(cm.GetResourceVersion())
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get resourceVersion from configmaps %#q in namespace %#q", cm.GetName(), cm.GetNamespace()), "stack", fmt.Sprintf("%#v", err))
				continue
			}

			if v <= highestResourceVersion {
				// no-op
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("no need to reconcile for the older resourceVersion %d", v))
				continue
			}

			configMap := resourceIndex{
				ResourceType: configMapType,
				Name:         cm.GetName(),
				Namespace:    cm.GetNamespace(),
			}

			var storedIndex map[appIndex]bool
			{
				v, ok := c.resourcesToApps.Load(configMap)
				if !ok {
					c.logger.Debugf(ctx, "cache missed configMap %#q in namespace %#q", configMap.Name, configMap.Namespace)
					continue
				}

				storedIndex, ok = v.(map[appIndex]bool)
				if !ok {
					c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("expected '%T', got '%T'", map[appIndex]bool{}, v), "stack", fmt.Sprintf("%#v", err))
					continue
				}
			}

			c.logger.Debugf(ctx, "listed apps depends on %#q configmap in namespace %#q", cm.Name, cm.Namespace)

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

				err = c.addAnnotation(ctx, &currentApp, cm.GetResourceVersion(), configMapType)
				if err != nil {
					c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to add annotation to app %#q in namespace %#q", app.Name, app.Namespace), "stack", fmt.Sprintf("%#v", err))
					continue
				}

				c.logger.Debugf(ctx, "triggered %#q app update in namespace %#q", app.Name, app.Namespace)

				c.event.Emit(ctx, &currentApp, "AppUpdated", "change to configmap %s/%s triggered an update", configMap.Namespace, configMap.Name)
			}
			c.appIndexMutex.RUnlock()
			c.logger.Debugf(ctx, "listed apps depends on %#q configmap in namespace %#q", cm.Name, cm.Namespace)
		}

		c.logger.Log("debug", "watch channel had been closed, reopening...")
	}
}

func (c *AppValueWatcher) addAnnotation(ctx context.Context, app *v1alpha1.App, latestResourceVersion string, resType resourceType) error {
	var versionAnnotation string
	{
		if resType == configMapType {
			versionAnnotation = annotation.AppOperatorLatestConfigMapVersion
		} else {
			versionAnnotation = annotation.AppOperatorLatestSecretVersion
		}
	}

	var modifiedApp v1alpha1.App

	err := c.k8sClient.CtrlClient().Get(
		ctx,
		types.NamespacedName{Name: app.Name, Namespace: app.Namespace},
		&modifiedApp,
	)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(modifiedApp.GetAnnotations()) == 0 {
		modifiedApp.Annotations = map[string]string{}
	}

	annotations := modifiedApp.Annotations
	annotations[versionAnnotation] = latestResourceVersion
	modifiedApp.Annotations = annotations

	err = c.k8sClient.CtrlClient().Patch(ctx, &modifiedApp, client.MergeFrom(app))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func getResourceVersion(resourceVersion string) (uint64, error) {
	r, err := strconv.ParseUint(resourceVersion, 0, 64)
	if err != nil {
		return 0, microerror.Mask(err)
	}

	return r, nil
}

// toConfigMap converts the input into a ConfigMap.
func toConfigMap(v interface{}) (*corev1.ConfigMap, error) {
	if v == nil {
		return &corev1.ConfigMap{}, nil
	}

	configMap, ok := v.(*corev1.ConfigMap)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.ConfigMap{}, v)
	}

	return configMap, nil
}
