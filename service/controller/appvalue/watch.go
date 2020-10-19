package appvalue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/giantswarm/app-operator/v2/pkg/annotation"
)

func (c *AppValue) watch(ctx context.Context) {
	var latestResourceVersion string

	for {
		lo := metav1.ListOptions{
			LabelSelector: c.selector.String(),
		}

		// Found the highest resourceVersion in cofigMaps CRs
		cms, err := c.k8sClient.K8sClient().CoreV1().ConfigMaps("").List(ctx, lo)
		if err != nil {
			panic(err)
		}

		for _, cm := range cms.Items {
			if latestResourceVersion < cm.ResourceVersion {
				latestResourceVersion = cm.ResourceVersion
			}
		}

		c.logger.Log("debug", fmt.Sprintf("starting ResourceVersion is %s", latestResourceVersion))

		lo = metav1.ListOptions{
			LabelSelector: c.selector.String(),
		}

		res, err := c.k8sClient.K8sClient().CoreV1().ConfigMaps("").Watch(ctx, lo)
		if err != nil {
			panic(err)
		}

		for r := range res.ResultChan() {
			if r.Type == watch.Bookmark || r.Type == watch.Error {
				// no-op for unsupported events
				continue
			}

			cm, err := toConfigMap(r.Object)
			if err != nil {
				panic(err)
			}

			if latestResourceVersion > cm.GetResourceVersion() {
				// no-op for the older events
				continue
			}

			configMap := configMapIndex{
				Name:      cm.GetName(),
				Namespace: cm.GetNamespace(),
			}

			var storedIndex map[appIndex]bool
			{
				v, ok := c.configMapToApps.Load(configMap)
				if !ok {
					c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("cache missed configMap %#q in namespace %#q", configMap.Name, configMap.Namespace))
					continue
				}

				storedIndex, ok = v.(map[appIndex]bool)
				if !ok {
					panic(fmt.Sprintf("expected '%T', got '%T'", map[appIndex]bool{}, v))
				}
			}

			for app := range storedIndex {
				c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("triggering %#q app updating in namespace %#q", app.Name, app.Namespace))

				err := c.addAnnotation(ctx, app, cm.GetResourceVersion())
				if err != nil {
					c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to add an annotation into app %#q in namespace %#q", app.Name, app.Namespace), "stack", fmt.Sprintf("%#v", err))
					continue
				}

				c.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("triggered %#q app updating in namespace %#q", app.Name, app.Namespace))
			}
		}

		c.logger.Log("debug", "watch channel had been closed, reopening...")
	}
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

func (c *AppValue) addAnnotation(ctx context.Context, app appIndex, latestResourceVersion string) error {
	versionAnnotation := fmt.Sprintf("%s/%s", annotation.AppOperatorPrefix, annotation.LatestConfigMapVersion)

	currentApp, err := c.k8sClient.G8sClient().ApplicationV1alpha1().Apps(app.Namespace).Get(ctx, app.Name, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	patches := []patch{}

	if len(currentApp.GetAnnotations()) == 0 {
		patches = append(patches, patch{
			Op:    "add",
			Path:  "/metadata/annotations",
			Value: map[string]string{},
		})
	}

	patches = append(patches, patch{
		Op:    "add",
		Path:  fmt.Sprintf("/metadata/annotations/%s", replaceToEscape(versionAnnotation)),
		Value: latestResourceVersion,
	})

	bytes, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.k8sClient.G8sClient().ApplicationV1alpha1().Apps(app.Namespace).Patch(ctx, app.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
