package appvalue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	k8slabel "github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace = "org-acme"
	valuesA       = "my-app-values-a"
	valuesB       = "my-app-values-b"
)

func testApp(userConfigMap string) *v1alpha1.App {
	return &v1alpha1.App{
		TypeMeta:   metav1.TypeMeta{APIVersion: "application.giantswarm.io/v1alpha1", Kind: "App"},
		ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: testNamespace},
		Spec: v1alpha1.AppSpec{
			Catalog:   "giantswarm",
			Name:      "my-app",
			Namespace: "my-app",
			UserConfig: v1alpha1.AppSpecUserConfig{
				ConfigMap: v1alpha1.AppSpecUserConfigConfigMap{Name: userConfigMap, Namespace: testNamespace},
			},
		},
	}
}

func valuesConfigMap(name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace}}
}

func configMapResource(name string) resourceIndex {
	return resourceIndex{ResourceType: configMapType, Name: name, Namespace: testNamespace}
}

// newCacheTestWatcher returns a watcher whose clients hold the giantswarm
// Catalog and the two values ConfigMaps.
func newCacheTestWatcher(t *testing.T, dyn dynamic.Interface) *AppValueWatcher {
	t.Helper()

	s := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("adding scheme: %v", err)
	}

	catalog := &v1alpha1.Catalog{ObjectMeta: metav1.ObjectMeta{Name: "giantswarm", Namespace: "giantswarm"}}

	return &AppValueWatcher{
		event: nopRecorder{},
		k8sClient: k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
			CtrlClient: fake.NewClientBuilder().WithScheme(s).WithObjects(catalog).Build(),
			DynClient:  dyn,
			K8sClient:  clientgofake.NewClientset(valuesConfigMap(valuesA), valuesConfigMap(valuesB)),
		}),
		logger:           microloggertest.New(),
		selector:         labels.Everything(),
		indexedResources: map[appIndex][]resourceIndex{},
	}
}

func isIndexed(c *AppValueWatcher, name string) bool {
	_, ok := c.resourcesToApps.Load(configMapResource(name))
	return ok
}

func isLabelled(t *testing.T, k8s kubernetes.Interface, name string) bool {
	t.Helper()

	cm, err := k8s.CoreV1().ConfigMaps(testNamespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("getting configmap %s: %v", name, err)
	}

	_, ok := cm.GetLabels()[k8slabel.AppOperatorWatching]
	return ok
}

// Test_addCache_updateDropsOldReference pins that an App that switches its
// values ConfigMap leaves the old one in the index. The old one keeps its
// watching label: another app-operator instance may still read it.
func Test_addCache_updateDropsOldReference(t *testing.T) {
	ctx := context.Background()
	c := newCacheTestWatcher(t, nil)

	if err := c.addCache(ctx, *testApp(valuesA), watch.Added); err != nil {
		t.Fatalf("adding app: %v", err)
	}
	if !isIndexed(c, valuesA) || !isLabelled(t, c.k8sClient.K8sClient(), valuesA) {
		t.Fatalf("%s is not indexed and labelled after the add", valuesA)
	}

	if err := c.addCache(ctx, *testApp(valuesB), watch.Modified); err != nil {
		t.Fatalf("updating app: %v", err)
	}
	if isIndexed(c, valuesA) {
		t.Fatalf("%s is still indexed after the app switched to %s", valuesA, valuesB)
	}
	if !isLabelled(t, c.k8sClient.K8sClient(), valuesA) {
		t.Fatalf("%s lost its watching label, which other app-operator instances share", valuesA)
	}
	if !isIndexed(c, valuesB) || !isLabelled(t, c.k8sClient.K8sClient(), valuesB) {
		t.Fatalf("%s is not indexed and labelled after the update", valuesB)
	}
}

// Test_addCache_deleteWithoutCatalog pins that a deleted App leaves the index
// although its Catalog is gone already, as in a cluster teardown.
func Test_addCache_deleteWithoutCatalog(t *testing.T) {
	ctx := context.Background()
	c := newCacheTestWatcher(t, nil)

	if err := c.addCache(ctx, *testApp(valuesA), watch.Added); err != nil {
		t.Fatalf("adding app: %v", err)
	}

	catalog := &v1alpha1.Catalog{ObjectMeta: metav1.ObjectMeta{Name: "giantswarm", Namespace: "giantswarm"}}
	if err := c.k8sClient.CtrlClient().Delete(ctx, catalog, &ctrlclient.DeleteOptions{}); err != nil {
		t.Fatalf("deleting catalog: %v", err)
	}

	if err := c.addCache(ctx, *testApp(valuesA), watch.Deleted); err != nil {
		t.Fatalf("deleting app: %v", err)
	}
	if isIndexed(c, valuesA) {
		t.Fatalf("%s is still indexed after its app was deleted", valuesA)
	}
	if len(c.indexedResources) != 0 {
		t.Fatalf("the deleted app is still in the reverse index: %v", c.indexedResources)
	}
}

// Test_addCache_deleteWithConfigMapGone pins that a deleted App leaves the
// index although its values ConfigMap was deleted first, as when Flux prunes
// both.
func Test_addCache_deleteWithConfigMapGone(t *testing.T) {
	ctx := context.Background()
	c := newCacheTestWatcher(t, nil)

	if err := c.addCache(ctx, *testApp(valuesA), watch.Added); err != nil {
		t.Fatalf("adding app: %v", err)
	}

	if err := c.k8sClient.K8sClient().CoreV1().ConfigMaps(testNamespace).Delete(ctx, valuesA, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("deleting configmap: %v", err)
	}

	if err := c.addCache(ctx, *testApp(valuesA), watch.Deleted); err != nil {
		t.Fatalf("deleting app: %v", err)
	}
	if isIndexed(c, valuesA) {
		t.Fatalf("%s is still indexed after its app was deleted", valuesA)
	}
}

// appListWatch serves the Apps of a fake dynamic client. Recent client-go
// informers list through a streaming watch (sendInitialEvents), which the fake
// cannot serve; refusing it makes the informer fall back to List.
func appListWatch(dyn dynamic.Interface) cache.ListerWatcher {
	return &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return dyn.Resource(appResource).Namespace(metav1.NamespaceAll).List(ctx, options)
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			if options.SendInitialEvents != nil && *options.SendInitialEvents {
				return nil, errors.New("streaming list not supported by the fake client")
			}
			return dyn.Resource(appResource).Namespace(metav1.NamespaceAll).Watch(ctx, options)
		},
	}
}

func newAppDynClient(t *testing.T, objects ...runtime.Object) dynamic.Interface {
	t.Helper()

	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{appResource: "AppList"},
		objects...,
	)
}

// Test_runAppInformer pins that the App informer indexes the ConfigMap an App
// reads its values from.
func Test_runAppInformer(t *testing.T) {
	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(testApp(valuesA))
	if err != nil {
		t.Fatalf("converting app: %v", err)
	}

	dyn := newAppDynClient(t, &unstructured.Unstructured{Object: content})
	c := newCacheTestWatcher(t, dyn)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		c.runAppInformer(ctx, appListWatch(dyn))
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()

	err = wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 5*time.Second, true, func(context.Context) (bool, error) {
		return isIndexed(c, valuesA), nil
	})
	if err != nil {
		t.Fatalf("the app's values ConfigMap was not indexed: %v", err)
	}
}

// Test_runAppInformer_stopsWithContext pins that the App watch ends when its
// context is cancelled, where the old loop kept retrying.
func Test_runAppInformer_stopsWithContext(t *testing.T) {
	dyn := newAppDynClient(t)
	c := newCacheTestWatcher(t, dyn)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		c.runAppInformer(ctx, appListWatch(dyn))
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runAppInformer did not return after its context was cancelled")
	}
}

// Test_stripApp pins that the cached Apps keep their spec and identity only.
func Test_stripApp(t *testing.T) {
	u := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":          "my-app",
			"namespace":     testNamespace,
			"managedFields": []interface{}{map[string]interface{}{"manager": "kubectl"}},
		},
		"spec":   map[string]interface{}{"catalog": "giantswarm"},
		"status": map[string]interface{}{"release": map[string]interface{}{"status": "deployed"}},
	}}

	obj, err := stripApp(u)
	if err != nil {
		t.Fatalf("stripApp: %v", err)
	}

	got := obj.(*unstructured.Unstructured)
	if _, ok := got.Object["status"]; ok {
		t.Fatal("stripApp kept the status")
	}
	if _, ok, _ := unstructured.NestedFieldNoCopy(got.Object, "metadata", "managedFields"); ok {
		t.Fatal("stripApp kept the managedFields")
	}
	if got.GetName() != "my-app" || got.GetNamespace() != testNamespace {
		t.Fatalf("stripApp lost the identity: %v", got.Object["metadata"])
	}
	if catalog, _, _ := unstructured.NestedString(got.Object, "spec", "catalog"); catalog != "giantswarm" {
		t.Fatalf("stripApp lost the spec: %v", got.Object["spec"])
	}
}
