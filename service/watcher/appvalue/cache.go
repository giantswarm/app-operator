package appvalue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v8/pkg/key"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// appResyncPeriod is how often every App is indexed and its ConfigMaps and
// Secrets labelled again. It retries Apps whose catalog or values could not be
// read yet, and labels a ConfigMap again whose label another app-operator
// instance removed. The watch reopens that used to rebuild the index did the
// same.
const appResyncPeriod = 30 * time.Minute

var appResource = schema.GroupVersionResource{Group: "application.giantswarm.io", Version: "v1alpha1", Resource: "apps"}

// buildCache links every App to the ConfigMaps and Secrets it reads its values
// from, and labels those for watching.
//
// It runs an informer, like the ConfigMap and Secret watches: it backs off on
// errors, stops with ctx, and resumes its watch or relists when the watch
// expires. A relist reports Apps deleted while the watch was down as
// deletions, so the index is updated in place and never rebuilt. Rebuilding it
// lost every ConfigMap or Secret change that arrived before the App replay had
// filled it again, and replaced the index while those handlers read it.
func (c *AppValueWatcher) buildCache(ctx context.Context) {
	lw := &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = c.selector.String()
			return c.k8sClient.DynClient().Resource(appResource).Namespace(metav1.NamespaceAll).List(ctx, options)
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = c.selector.String()
			return c.k8sClient.DynClient().Resource(appResource).Namespace(metav1.NamespaceAll).Watch(ctx, options)
		},
	}

	c.runAppInformer(ctx, lw)
}

func (c *AppValueWatcher) runAppInformer(ctx context.Context, lw cache.ListerWatcher) {
	informer := cache.NewSharedIndexInformer(lw, &unstructured.Unstructured{}, appResyncPeriod, cache.Indexers{})

	// The handlers read the spec, name and namespace of an App only.
	err := informer.SetTransform(stripApp)
	if err != nil {
		c.logger.Errorf(ctx, err, "failed to set the transform for apps")
		return
	}

	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.onAppEvent(ctx, obj, watch.Added)
		},
		UpdateFunc: func(_, newObj interface{}) {
			c.onAppEvent(ctx, newObj, watch.Modified)
		},
		DeleteFunc: func(obj interface{}) {
			if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = tombstone.Obj
			}
			c.onAppEvent(ctx, obj, watch.Deleted)
		},
	})
	if err != nil {
		c.logger.Errorf(ctx, err, "failed to add event handler for apps")
		return
	}

	informer.Run(ctx.Done())
}

// stripApp drops what the App handlers never read from the cached Apps.
func stripApp(obj interface{}) (interface{}, error) {
	if u, ok := obj.(*unstructured.Unstructured); ok {
		unstructured.RemoveNestedField(u.Object, "status")
		unstructured.RemoveNestedField(u.Object, "metadata", "managedFields")
	}

	return obj, nil
}

func (c *AppValueWatcher) onAppEvent(ctx context.Context, obj interface{}, eventType watch.EventType) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		c.logger.Errorf(ctx, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &unstructured.Unstructured{}, obj), "failed to read app")
		return
	}

	app := &v1alpha1.App{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), app)
	if err != nil {
		c.logger.Errorf(ctx, err, "failed to convert unstructured object %#v to app", u)
		return
	}

	err = c.addCache(ctx, *app, eventType)
	if err != nil {
		c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to reconcile app CR %#q", app.Name), "stack", fmt.Sprintf("%#v", err))
	}
}

func (c *AppValueWatcher) addCache(ctx context.Context, cr v1alpha1.App, eventType watch.EventType) error {
	app := appIndex{
		Name:      cr.GetName(),
		Namespace: cr.GetNamespace(),
	}

	if c.indexedResources == nil {
		c.indexedResources = map[appIndex][]resourceIndex{}
	}

	switch eventType {
	case watch.Added, watch.Modified:
		resources, err := c.appResources(ctx, cr)
		if err != nil {
			// The App keeps the resources it was indexed with; the next
			// resync retries it.
			return microerror.Mask(err)
		}

		for _, resource := range resources {
			// Label it for watching. A label that cannot be set yet, for
			// example on a ConfigMap that does not exist yet, is set again
			// on the next resync.
			err := c.addLabel(ctx, resource)
			if err != nil {
				c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to add label to %#q %#q in namespace %#q", resource.ResourceType, resource.Name, resource.Namespace), "stack", fmt.Sprintf("%#v", err))
			}

			c.linkApp(app, resource)
		}

		// Drop what the App no longer reads from. The watching label stays:
		// every app-operator instance shares it, and this one cannot tell
		// whether another one still reads the resource.
		for _, resource := range c.indexedResources[app] {
			if !containsResource(resources, resource) {
				c.unlinkApp(ctx, app, resource, false)
			}
		}

		c.indexedResources[app] = resources

	case watch.Deleted:
		// The resources come from the index, not from the App: its catalog
		// may be gone already, as in a cluster teardown.
		for _, resource := range c.indexedResources[app] {
			c.unlinkApp(ctx, app, resource, true)
		}

		delete(c.indexedResources, app)

	default:
		// no-op for unsupported events
	}

	return nil
}

// appResources lists the ConfigMaps and Secrets the App reads its values from.
func (c *AppValueWatcher) appResources(ctx context.Context, cr v1alpha1.App) ([]resourceIndex, error) {
	resources := []resourceIndex{}

	catalog, err := c.findCatalog(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if key.CatalogConfigMapName(*catalog) != "" {
		resources = append(resources, resourceIndex{
			ResourceType: configMapType,
			Name:         key.CatalogConfigMapName(*catalog),
			Namespace:    key.CatalogConfigMapNamespace(*catalog),
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

	if key.CatalogSecretName(*catalog) != "" {
		resources = append(resources, resourceIndex{
			ResourceType: secretType,
			Name:         key.CatalogSecretName(*catalog),
			Namespace:    key.CatalogSecretNamespace(*catalog),
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

	// Watch extra configs as well
	for _, extraConfig := range key.ExtraConfigs(cr) {
		var kind resourceType
		switch extraConfig.Kind {
		case "secret":
			kind = secretType
		default:
			kind = configMapType
		}

		resources = append(resources, resourceIndex{
			ResourceType: kind,
			Name:         extraConfig.Name,
			Namespace:    extraConfig.Namespace,
		})
	}

	return resources, nil
}

// linkApp records that the App reads its values from the resource.
func (c *AppValueWatcher) linkApp(app appIndex, resource resourceIndex) {
	c.appIndexMutex.Lock()
	defer c.appIndexMutex.Unlock()

	v, ok := c.resourcesToApps.Load(resource)
	if !ok {
		c.resourcesToApps.Store(resource, map[appIndex]bool{app: true})
		return
	}

	storedIndex, ok := v.(map[appIndex]bool)
	if !ok {
		c.resourcesToApps.Store(resource, map[appIndex]bool{app: true})
		return
	}

	storedIndex[app] = true
}

// unlinkApp removes the App from the resource's Apps. A resource no App reads
// from any more leaves the index, and with removeLabel loses its watching
// label too; one that is gone already only leaves the index.
func (c *AppValueWatcher) unlinkApp(ctx context.Context, app appIndex, resource resourceIndex, removeLabel bool) {
	empty := false
	{
		c.appIndexMutex.Lock()
		v, ok := c.resourcesToApps.Load(resource)
		if ok {
			storedIndex, ok := v.(map[appIndex]bool)
			if ok {
				delete(storedIndex, app)
				empty = len(storedIndex) == 0
			} else {
				empty = true
			}
			if empty {
				c.resourcesToApps.Delete(resource)
			}
		}
		c.appIndexMutex.Unlock()
	}

	if !empty || !removeLabel {
		return
	}

	err := c.removeLabel(ctx, resource)
	if apierrors.IsNotFound(err) {
		return
	} else if err != nil {
		c.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to remove label from %#q %#q in namespace %#q", resource.ResourceType, resource.Name, resource.Namespace), "stack", fmt.Sprintf("%#v", err))
	}
}

func containsResource(resources []resourceIndex, resource resourceIndex) bool {
	for _, r := range resources {
		if r == resource {
			return true
		}
	}

	return false
}

func (c *AppValueWatcher) addLabel(ctx context.Context, resource resourceIndex) error {
	var currentLabels map[string]string
	{
		switch resource.ResourceType {
		case configMapType:
			currentCM, err := c.k8sClient.K8sClient().CoreV1().ConfigMaps(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			currentLabels = currentCM.GetLabels()
		case secretType:
			currentSecret, err := c.k8sClient.K8sClient().CoreV1().Secrets(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			currentLabels = currentSecret.GetLabels()
		default:
			return microerror.Maskf(wrongTypeError, "expected %T or %T but got %T", configMapType, secretType, resource.ResourceType)
		}
	}

	if _, ok := currentLabels[label.AppOperatorWatching]; ok {
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
		Path:  fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorWatching)),
		Value: "true",
	})

	bytes, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	switch resource.ResourceType {
	case configMapType:
		_, err = c.k8sClient.K8sClient().CoreV1().ConfigMaps(resource.Namespace).Patch(ctx, resource.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	case secretType:
		_, err = c.k8sClient.K8sClient().CoreV1().Secrets(resource.Namespace).Patch(ctx, resource.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (c *AppValueWatcher) findCatalog(ctx context.Context, cr v1alpha1.App) (*v1alpha1.Catalog, error) {
	var err error
	var namespaces []string
	{
		if cr.Spec.CatalogNamespace != "" {
			namespaces = []string{cr.Spec.CatalogNamespace}
		} else {
			namespaces = []string{metav1.NamespaceDefault, "giantswarm"}
		}
	}

	catalog := &v1alpha1.Catalog{}

	for _, namespace := range namespaces {
		err = c.k8sClient.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: key.CatalogName(cr), Namespace: namespace},
			catalog,
		)
		if apierrors.IsNotFound(err) {
			// no-op
			continue
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
		break
	}

	if catalog.GetName() == "" {
		return nil, microerror.Maskf(notFoundError, "catalog %#q", key.CatalogName(cr))
	}

	return catalog, nil
}

func (c *AppValueWatcher) removeLabel(ctx context.Context, resource resourceIndex) error {
	var currentLabels map[string]string
	{
		switch resource.ResourceType {
		case configMapType:
			currentCM, err := c.k8sClient.K8sClient().CoreV1().ConfigMaps(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			currentLabels = currentCM.GetLabels()
		case secretType:
			currentSecret, err := c.k8sClient.K8sClient().CoreV1().Secrets(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			currentLabels = currentSecret.GetLabels()
		default:
			return microerror.Maskf(wrongTypeError, "expected %T or %T but got %T", configMapType, secretType, resource.ResourceType)
		}
	}

	if _, ok := currentLabels[label.AppOperatorWatching]; !ok {
		// no-op
		return nil
	}

	patches := []patch{
		{
			Op:   "remove",
			Path: fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorWatching)),
		},
	}

	bytes, err := json.Marshal(patches)
	if err != nil {
		return microerror.Mask(err)
	}

	switch resource.ResourceType {
	case configMapType:
		_, err = c.k8sClient.K8sClient().CoreV1().ConfigMaps(resource.Namespace).Patch(ctx, resource.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	case secretType:
		_, err = c.k8sClient.K8sClient().CoreV1().Secrets(resource.Namespace).Patch(ctx, resource.Name, types.JSONPatchType, bytes, metav1.PatchOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func replaceToEscape(from string) string {
	return strings.ReplaceAll(from, "/", "~1")
}
