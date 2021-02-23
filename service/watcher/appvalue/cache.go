package appvalue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	pkglabel "github.com/giantswarm/app-operator/v4/pkg/label"
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
		c.resourcesToApps = sync.Map{}
	}

}

func (c *AppValueWatcher) addCache(ctx context.Context, cr v1alpha1.App, eventType watch.EventType) error {
	app := appIndex{
		Name:      cr.GetName(),
		Namespace: cr.GetNamespace(),
	}

	resources := []resourceIndex{}

	appCatalog, err := c.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogs().Get(ctx, key.CatalogName(cr), metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	if key.AppCatalogConfigMapName(*appCatalog) != "" {
		resources = append(resources, resourceIndex{
			ResourceType: configMapType,
			Name:         key.AppCatalogConfigMapName(*appCatalog),
			Namespace:    key.AppCatalogConfigMapNamespace(*appCatalog),
		})
	}

	if key.AppConfigMapName(cr) != "" {
		resources = append(resources, resourceIndex{
			ResourceType: configMapType,
			Name:         key.AppConfigMapName(cr),
			Namespace:    key.AppConfigMapNamespace(cr),
		})
	}

	if key.UserConfigMapName(cr) != "" {
		resources = append(resources, resourceIndex{
			ResourceType: configMapType,
			Name:         key.UserConfigMapName(cr),
			Namespace:    key.UserConfigMapNamespace(cr),
		})
	}

	if key.AppCatalogSecretName(*appCatalog) != "" {
		resources = append(resources, resourceIndex{
			ResourceType: secretType,
			Name:         key.AppCatalogSecretName(*appCatalog),
			Namespace:    key.AppCatalogSecretNamespace(*appCatalog),
		})
	}

	if key.AppSecretName(cr) != "" {
		resources = append(resources, resourceIndex{
			ResourceType: secretType,
			Name:         key.AppSecretName(cr),
			Namespace:    key.AppSecretNamespace(cr),
		})
	}

	if key.UserSecretName(cr) != "" {
		resources = append(resources, resourceIndex{
			ResourceType: secretType,
			Name:         key.UserSecretName(cr),
			Namespace:    key.UserSecretNamespace(cr),
		})
	}

	switch eventType {
	case watch.Added, watch.Modified:
		for _, resource := range resources {
			// First, put the watchUpdate label
			err := c.addLabel(ctx, resource)
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to add label to %#q %#q in namespace %#q", resource.ResourceType, resource.Name, resource.Namespace), "stack", fmt.Sprintf("%#v", err))
				continue
			}

			v, ok := c.resourcesToApps.Load(resource)
			if ok {
				storedAppIndex, ok := v.(map[appIndex]bool)
				if !ok {
					return microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", []appIndex{}, v)
				}

				storedAppIndex[app] = true
				c.resourcesToApps.Store(resource, storedAppIndex)
			} else {
				m := map[appIndex]bool{
					app: true,
				}
				c.resourcesToApps.Store(resource, m)
			}
		}

	case watch.Deleted:
		for _, resource := range resources {
			v, ok := c.resourcesToApps.Load(resource)
			if ok {
				storedIndex, ok := v.(map[appIndex]bool)
				if !ok {
					return microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", []appIndex{}, v)
				}

				delete(storedIndex, app)
				if len(storedIndex) == 0 {
					err := c.removeLabel(ctx, resource)
					if err != nil {
						c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to remove label from %#q %#q in namespace %#q", resource.ResourceType, resource.Name, resource.Namespace), "stack", fmt.Sprintf("%#v", err))
						continue
					}

					c.resourcesToApps.Delete(resource)
				} else {
					c.resourcesToApps.Store(resource, storedIndex)
				}
			}
		}

	default:
		// no-op for unsupported events
	}

	return nil
}

func (c *AppValueWatcher) addLabel(ctx context.Context, resource resourceIndex) error {
	var currentLabels map[string]string
	{
		if resource.ResourceType == configMapType {
			currentCM, err := c.k8sClient.K8sClient().CoreV1().ConfigMaps(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			currentLabels = currentCM.GetLabels()
		} else if resource.ResourceType == secretType {
			currentSecret, err := c.k8sClient.K8sClient().CoreV1().Secrets(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			currentLabels = currentSecret.GetLabels()
		} else {
			return microerror.Maskf(wrongTypeError, "expected %T or %T but got %T", configMapType, secretType, resource.ResourceType)
		}
	}

	if _, ok := currentLabels[pkglabel.Watching]; ok {
		// no-op
		return nil
	}

	patches := []patch{}

	if len(currentLabels) == 0 {
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

	if resource.ResourceType == configMapType {
		_, err = c.k8sClient.K8sClient().CoreV1().ConfigMaps(resource.Namespace).Patch(ctx, resource.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	} else if resource.ResourceType == secretType {
		_, err = c.k8sClient.K8sClient().CoreV1().Secrets(resource.Namespace).Patch(ctx, resource.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (c *AppValueWatcher) removeLabel(ctx context.Context, resource resourceIndex) error {
	var currentLabels map[string]string
	{
		if resource.ResourceType == configMapType {
			currentCM, err := c.k8sClient.K8sClient().CoreV1().ConfigMaps(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			currentLabels = currentCM.GetLabels()
		} else if resource.ResourceType == secretType {
			currentSecret, err := c.k8sClient.K8sClient().CoreV1().Secrets(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			currentLabels = currentSecret.GetLabels()
		} else {
			return microerror.Maskf(wrongTypeError, "expected %T or %T but got %T", configMapType, secretType, resource.ResourceType)
		}
	}

	if _, ok := currentLabels[pkglabel.Watching]; !ok {
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

	if resource.ResourceType == configMapType {
		_, err = c.k8sClient.K8sClient().CoreV1().ConfigMaps(resource.Namespace).Patch(ctx, resource.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	} else if resource.ResourceType == secretType {
		_, err = c.k8sClient.K8sClient().CoreV1().Secrets(resource.Namespace).Patch(ctx, resource.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func replaceToEscape(from string) string {
	return strings.Replace(from, "/", "~1", -1)
}
