package appvalue

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

	"github.com/giantswarm/app-operator/v2/pkg/annotation"
	"github.com/giantswarm/app-operator/v2/service/controller/app/key"
)

func (c *AppValue) buildCache(ctx context.Context) error {
	for {
		res, err := c.k8sClient.G8sClient().ApplicationV1alpha1().Apps("").Watch(ctx, metav1.ListOptions{})
		if err != nil {
			panic(err)
		}

		for r := range res.ResultChan() {
			cr, err := key.ToCustomResource(r.Object)
			if err != nil {
				panic(err)
			}

			err = c.addCache(ctx, cr, r.Type)
			if err != nil {
				c.logger.Log("level", "info", "message", "failed to reconcile an app CR", "stack", fmt.Sprintf("%#v", err))
			}
		}

		c.logger.Log("debug", "watch channel had been closed, reopening...")
		c.apps = sync.Map{}
	}

}

func (c *AppValue) addCache(ctx context.Context, cr v1alpha1.App, eventType watch.EventType) error {
	appIndex := index{
		Name:      cr.GetName(),
		Namespace: cr.GetNamespace(),
	}

	configMaps := []index{}
	if key.AppConfigMapName(cr) != "" {
		configMaps = append(configMaps, index{
			Name:      key.AppConfigMapName(cr),
			Namespace: key.AppConfigMapNamespace(cr),
		})
	}

	if key.UserConfigMapName(cr) != "" {
		configMaps = append(configMaps, index{
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
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to add a label into configmap %#q in namespace %#q", configMap.Name, configMap.Namespace), "stack", fmt.Sprintf("%#v", err))
				continue
			}

			v, ok := c.apps.Load(configMap)
			if ok {
				storedIndex, ok := v.(map[index]bool)
				if !ok {
					return microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", []index{}, v)
				}

				storedIndex[appIndex] = true
				c.apps.Store(configMap, storedIndex)
			} else {
				m := map[index]bool{
					appIndex: true,
				}
				c.apps.Store(configMap, m)
			}
		}

	case watch.Deleted:
		for _, configMap := range configMaps {
			v, ok := c.apps.Load(configMap)
			if ok {
				storedIndex, ok := v.(map[index]bool)
				if !ok {
					return microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", []index{}, v)
				}

				delete(storedIndex, appIndex)
				if len(storedIndex) == 0 {
					c.apps.Delete(configMap)
				} else {
					c.apps.Store(configMap, storedIndex)
				}
			}
		}

	default:
		c.logger.Log("debug", fmt.Sprintf("event %#q for app %#q is not supported", eventType, cr.Name))
	}

	return nil
}

func (c *AppValue) addLabel(ctx context.Context, cm index) error {
	watchUpdate := replaceToEscape(fmt.Sprintf("%s/%s", annotation.AppOperatorPrefix, annotation.WatchUpdate))

	currentCM, err := c.k8sClient.G8sClient().ApplicationV1alpha1().Charts(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	if _, ok := currentCM.GetLabels()[watchUpdate]; ok {
		// no-op
		return nil
	}

	patches := []patch{
		{
			Op:    "add",
			Path:  fmt.Sprintf("/metadata/annotations/%s", watchUpdate),
			Value: true,
		},
	}
	bytes, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.k8sClient.G8sClient().ApplicationV1alpha1().Charts(cm.Namespace).Patch(ctx, cm.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func replaceToEscape(from string) string {
	return strings.Replace(from, "/", "~1", -1)
}
