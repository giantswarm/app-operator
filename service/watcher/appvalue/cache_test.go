package appvalue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
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
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test_buildCache pins that the App informer indexes the ConfigMap an App
// reads its values from, and drops it again when the App is deleted.
func Test_buildCache(t *testing.T) {
	s := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("adding scheme: %v", err)
	}

	catalog := &v1alpha1.Catalog{ObjectMeta: metav1.ObjectMeta{Name: "giantswarm", Namespace: "giantswarm"}}
	userValues := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "my-app-user-values", Namespace: "org-acme"}}

	app := &v1alpha1.App{
		TypeMeta:   metav1.TypeMeta{APIVersion: "application.giantswarm.io/v1alpha1", Kind: "App"},
		ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "org-acme"},
		Spec: v1alpha1.AppSpec{
			Catalog:   "giantswarm",
			Name:      "my-app",
			Namespace: "my-app",
			UserConfig: v1alpha1.AppSpecUserConfig{
				ConfigMap: v1alpha1.AppSpecUserConfigConfigMap{Name: "my-app-user-values", Namespace: "org-acme"},
			},
		},
	}
	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(app)
	if err != nil {
		t.Fatalf("converting app: %v", err)
	}

	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{appResource: "AppList"},
		&unstructured.Unstructured{Object: content},
	)

	c := &AppValueWatcher{
		event: nopRecorder{},
		k8sClient: k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
			CtrlClient: fake.NewClientBuilder().WithScheme(s).WithObjects(catalog).Build(),
			DynClient:  dyn,
			K8sClient:  clientgofake.NewClientset(userValues),
		}),
		logger:   microloggertest.New(),
		selector: labels.Everything(),
	}

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

	resource := resourceIndex{ResourceType: configMapType, Name: "my-app-user-values", Namespace: "org-acme"}
	indexed := func(want bool) func(context.Context) (bool, error) {
		return func(context.Context) (bool, error) {
			_, ok := c.resourcesToApps.Load(resource)
			return ok == want, nil
		}
	}

	if err := wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 5*time.Second, true, indexed(true)); err != nil {
		t.Fatalf("the app's user values ConfigMap was not indexed: %v", err)
	}

	err = dyn.Resource(appResource).Namespace("org-acme").Delete(ctx, "my-app", metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("deleting app: %v", err)
	}

	if err := wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 5*time.Second, true, indexed(false)); err != nil {
		t.Fatalf("the ConfigMap of the deleted app is still indexed: %v", err)
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

// Test_buildCache_stopsWithContext pins that the App watch ends when its
// context is cancelled, where the old loop kept retrying.
func Test_buildCache_stopsWithContext(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{appResource: "AppList"},
	)

	c := &AppValueWatcher{
		event:     nopRecorder{},
		k8sClient: k8sclienttest.NewClients(k8sclienttest.ClientsConfig{DynClient: dyn}),
		logger:    microloggertest.New(),
		selector:  labels.Everything(),
	}

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
		t.Fatal("buildCache did not return after its context was cancelled")
	}
}
