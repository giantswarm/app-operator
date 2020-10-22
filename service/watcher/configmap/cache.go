package configmap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	pkglabel "github.com/giantswarm/app-operator/v2/pkg/label"
	"github.com/giantswarm/app-operator/v2/service/controller/key"
)

func (c *AppValueWatcher) buildCache(ctx context.Context) {
	for {
		lo := metav1.ListOptions{
			LabelSelector: c.selector.String(),
		}

		res, err := c.k8sClient.G8sClient().ApplicationV1alpha1().Apps("").Watch(ctx, lo)
		if err != nil {
			c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get apps with label %#q", c.selector.String()), "stack", fmt.Sprintf("%#v", err))
			continue
		}

		for r := range res.ResultChan() {
			cr, err := key.ToApp(r.Object)
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", "failed to convert app object", "stack", fmt.Sprintf("%#v", err))
				continue
			}

			err = c.addCache(ctx, cr, r.Type)
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to reconcile app CR %#q", cr.Name), "stack", fmt.Sprintf("%#v", err))
			}
		}

		c.logger.LogCtx(ctx, "debug", "watch channel has been closed, reopening...")
		c.configMapToApps = sync.Map{}
	}

}

func (c *AppValueWatcher) addCache(ctx context.Context, cr v1alpha1.App, eventType watch.EventType) error {
	app := appIndex{
		Name:      cr.GetName(),
		Namespace: cr.GetNamespace(),
	}

	configMaps := []configMapIndex{}

	appCatalog, err := c.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogs().Get(ctx, key.CatalogName(cr), metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	if key.AppCatalogConfigMapName(*appCatalog) != "" {
		configMaps = append(configMaps, configMapIndex{
			Name:      key.AppCatalogConfigMapName(*appCatalog),
			Namespace: key.AppCatalogConfigMapNamespace(*appCatalog),
		})
	}

	if key.AppConfigMapName(cr) != "" {
		configMaps = append(configMaps, configMapIndex{
			Name:      key.AppConfigMapName(cr),
			Namespace: key.AppConfigMapNamespace(cr),
		})
	}

	if key.UserConfigMapName(cr) != "" {
		configMaps = append(configMaps, configMapIndex{
			Name:      key.UserConfigMapName(cr),
			Namespace: key.UserConfigMapNamespace(cr),
		})
	}

	switch eventType {
	case watch.Added, watch.Modified:
		for _, configMap := range configMaps {
			// First, put the watchUpdate label
			err := c.addLabel(ctx, configMap)
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to add label to configmap %#q in namespace %#q", configMap.Name, configMap.Namespace), "stack", fmt.Sprintf("%#v", err))
				continue
			}

			v, ok := c.configMapToApps.Load(configMap)
			if ok {
				storedAppIndex, ok := v.(map[appIndex]bool)
				if !ok {
					return microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", []appIndex{}, v)
				}

				storedAppIndex[app] = true
				c.configMapToApps.Store(configMap, storedAppIndex)
			} else {
				m := map[appIndex]bool{
					app: true,
				}
				c.configMapToApps.Store(configMap, m)
			}
		}

	case watch.Deleted:
		for _, configMap := range configMaps {
			v, ok := c.configMapToApps.Load(configMap)
			if ok {
				storedIndex, ok := v.(map[appIndex]bool)
				if !ok {
					return microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", []appIndex{}, v)
				}

				delete(storedIndex, app)
				if len(storedIndex) == 0 {
					err := c.removeLabel(ctx, configMap)
					if err != nil {
						c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to remove label from configmap %#q in namespace %#q", configMap.Name, configMap.Namespace), "stack", fmt.Sprintf("%#v", err))
						continue
					}

					c.configMapToApps.Delete(configMap)
				} else {
					c.configMapToApps.Store(configMap, storedIndex)
				}
			}
		}

	default:
		// no-op for unsupported events
	}

	return nil
}

func (c *AppValueWatcher) addLabel(ctx context.Context, cm configMapIndex) error {
	currentCM, err := c.k8sClient.K8sClient().CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	if _, ok := currentCM.GetLabels()[pkglabel.Watching]; ok {
		// no-op
		return nil
	}

	patches := []patch{}

	if len(currentCM.GetLabels()) == 0 {
		patches = append(patches, patch{
			Op:    "add",
			Path:  "/metadata/labels",
			Value: map[string]string{},
		})
	}

	patches = append(patches, patch{
		Op:    "add",
		Path:  fmt.Sprintf("/metadata/labels/%s", replaceToEscape(pkglabel.Watching)),
		Value: "true",
	})

	bytes, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.k8sClient.K8sClient().CoreV1().ConfigMaps(cm.Namespace).Patch(ctx, cm.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (c *AppValueWatcher) removeLabel(ctx context.Context, cm configMapIndex) error {
	currentCM, err := c.k8sClient.K8sClient().CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	if _, ok := currentCM.GetLabels()[pkglabel.Watching]; !ok {
		// no-op
		return nil
	}

	patches := []patch{
		{
			Op:   "remove",
			Path: fmt.Sprintf("/metadata/labels/%s", replaceToEscape(pkglabel.Watching)),
		},
	}

	bytes, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.k8sClient.K8sClient().CoreV1().ConfigMaps(cm.Namespace).Patch(ctx, cm.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func replaceToEscape(from string) string {
	return strings.Replace(from, "/", "~1", -1)
}