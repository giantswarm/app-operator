package appvalue

import (
	"context"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deletedVersionSuffix marks the version annotation written for a deleted
// ConfigMap or Secret. The annotation is only a trigger; nothing parses it.
const deletedVersionSuffix = "-deleted"

// runInformer watches the ConfigMaps or Secrets listed by lw and triggers an
// update of every App that depends on one of them when it changes or is
// deleted.
//
// An informer resumes its watch from the last resourceVersion it has seen.
// When that is too old, it relists and reports every object whose
// resourceVersion changed in the meantime as an update, so a change made while
// the watch was being reopened is not lost. It also backs off on errors.
//
// The objects of the initial list arrive as adds and are ignored, as they are
// not changes. A later add triggers the depending Apps: an object that was
// deleted and created again under the same name (as kubectl replace --force
// does) must re-render them once it exists again. An object buildCache just
// labelled usually triggers its Apps once more, which is harmless.
func (c *AppValueWatcher) runInformer(ctx context.Context, lw cache.ListerWatcher, exampleObject runtime.Object, resType resourceType) {
	informer := cache.NewSharedIndexInformer(lw, exampleObject, 0, cache.Indexers{})

	// Only name, namespace and resourceVersion are read. Every app-operator
	// instance watches all labelled ConfigMaps of the management cluster, so
	// keep their data, and the data of Secrets, out of its memory. Annotations
	// go too: kubectl.kubernetes.io/last-applied-configuration holds the whole
	// object a client-side kubectl apply wrote, data included.
	err := informer.SetTransform(stripData)
	if err != nil {
		c.logger.Errorf(ctx, err, "failed to set the transform for %ss", resType)
		return
	}

	_, err = informer.AddEventHandler(cache.ResourceEventHandlerDetailedFuncs{
		AddFunc: func(obj interface{}, isInInitialList bool) {
			if isInInitialList {
				return
			}
			c.onAdd(ctx, obj, resType)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.onUpdate(ctx, oldObj, newObj, resType)
		},
		DeleteFunc: func(obj interface{}) {
			c.onDelete(ctx, obj, resType)
		},
	})
	if err != nil {
		c.logger.Errorf(ctx, err, "failed to add event handler for %ss", resType)
		return
	}

	informer.Run(ctx.Done())
}

// stripData drops what the watcher never reads from the cached objects.
func stripData(obj interface{}) (interface{}, error) {
	switch o := obj.(type) {
	case *corev1.ConfigMap:
		o.Data = nil
		o.BinaryData = nil
		o.Annotations = nil
		o.ManagedFields = nil
	case *corev1.Secret:
		o.Data = nil
		o.StringData = nil
		o.Annotations = nil
		o.ManagedFields = nil
	}

	return obj, nil
}

func (c *AppValueWatcher) onAdd(ctx context.Context, obj interface{}, resType resourceType) {
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		c.logger.Errorf(ctx, err, "failed to read the metadata of the added %s", resType)
		return
	}

	c.triggerApps(ctx, resourceIndex{
		ResourceType: resType,
		Name:         objMeta.GetName(),
		Namespace:    objMeta.GetNamespace(),
	}, objMeta.GetResourceVersion())
}

func (c *AppValueWatcher) onUpdate(ctx context.Context, oldObj, newObj interface{}, resType resourceType) {
	oldMeta, err := meta.Accessor(oldObj)
	if err != nil {
		c.logger.Errorf(ctx, err, "failed to read the metadata of the previous %s", resType)
		return
	}

	newMeta, err := meta.Accessor(newObj)
	if err != nil {
		c.logger.Errorf(ctx, err, "failed to read the metadata of the %s", resType)
		return
	}

	// A relist reports unchanged objects as updates too.
	if oldMeta.GetResourceVersion() == newMeta.GetResourceVersion() {
		return
	}

	c.triggerApps(ctx, resourceIndex{
		ResourceType: resType,
		Name:         newMeta.GetName(),
		Namespace:    newMeta.GetNamespace(),
	}, newMeta.GetResourceVersion())
}

func (c *AppValueWatcher) onDelete(ctx context.Context, obj interface{}, resType resourceType) {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}

	objMeta, err := meta.Accessor(obj)
	if err != nil {
		c.logger.Errorf(ctx, err, "failed to read the metadata of the deleted %s", resType)
		return
	}

	// A deletion seen only on a relist arrives as a tombstone that carries the
	// last resourceVersion the informer cached, which the App may already
	// carry from the update before. Mark it as a deletion, so the annotation
	// always changes and the App is rendered again without the object.
	c.triggerApps(ctx, resourceIndex{
		ResourceType: resType,
		Name:         objMeta.GetName(),
		Namespace:    objMeta.GetNamespace(),
	}, objMeta.GetResourceVersion()+deletedVersionSuffix)
}

// triggerApps annotates every App that depends on the given resource with its
// latest resourceVersion, which makes the App controller reconcile the App.
func (c *AppValueWatcher) triggerApps(ctx context.Context, resource resourceIndex, resourceVersion string) {
	// Copy the Apps under the lock and release it before the API calls, so
	// indexing does not wait for them. The lookup is under the lock too, so it
	// sees the index the App informer last wrote.
	var apps []appIndex
	{
		c.appIndexMutex.RLock()
		v, ok := c.resourcesToApps.Load(resource)
		if ok {
			storedIndex, isIndex := v.(map[appIndex]bool)
			if !isIndex {
				c.appIndexMutex.RUnlock()
				c.logger.Errorf(ctx, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[appIndex]bool{}, v), "failed to read the apps of %s %#q in namespace %#q", resource.ResourceType, resource.Name, resource.Namespace)
				return
			}
			for app := range storedIndex {
				apps = append(apps, app)
			}
		}
		c.appIndexMutex.RUnlock()

		if !ok {
			c.logger.Debugf(ctx, "cache missed %s %#q in namespace %#q", resource.ResourceType, resource.Name, resource.Namespace)
			return
		}
	}

	c.logger.Debugf(ctx, "listed apps depends on %#q %s in namespace %#q", resource.Name, resource.ResourceType, resource.Namespace)

	for _, app := range apps {
		c.logger.Debugf(ctx, "triggering %#q app update in namespace %#q", app.Name, app.Namespace)

		var currentApp v1alpha1.App
		err := c.k8sClient.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: app.Name, Namespace: app.Namespace},
			&currentApp,
		)
		if err != nil {
			c.logger.Errorf(ctx, err, "cannot fetch app CR %s/%s", app.Namespace, app.Name)
			continue
		}

		err = c.addAnnotation(ctx, &currentApp, resourceVersion, resource.ResourceType)
		if err != nil {
			c.logger.Errorf(ctx, err, "failed to add annotation to app %#q in namespace %#q", app.Name, app.Namespace)
			continue
		}

		c.logger.Debugf(ctx, "triggered %#q app update in namespace %#q", app.Name, app.Namespace)

		c.event.Emit(ctx, &currentApp, "AppUpdated", "change to %s %s/%s triggered an update", resource.ResourceType, resource.Namespace, resource.Name)
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
