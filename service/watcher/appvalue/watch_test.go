package appvalue

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/micrologger/microloggertest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type nopRecorder struct{}

func (nopRecorder) Emit(context.Context, runtime.Object, string, string, ...interface{}) {}

func newTestWatcher(t *testing.T) *AppValueWatcher {
	t.Helper()

	s := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("adding scheme: %v", err)
	}

	app := &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: "org-acme"},
	}

	clients := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
		CtrlClient: fake.NewClientBuilder().WithScheme(s).WithObjects(app).Build(),
		K8sClient:  clientgofake.NewClientset(),
	})

	c := &AppValueWatcher{
		event:     nopRecorder{},
		k8sClient: clients,
		logger:    microloggertest.New(),
	}
	c.resourcesToApps.Store(
		resourceIndex{ResourceType: configMapType, Name: "my-app-user-values", Namespace: "org-acme"},
		map[appIndex]bool{{Name: "my-app", Namespace: "org-acme"}: true},
	)

	return c
}

func latestConfigMapVersion(t *testing.T, c *AppValueWatcher) string {
	t.Helper()

	var app v1alpha1.App
	err := c.k8sClient.CtrlClient().Get(context.Background(), types.NamespacedName{Name: "my-app", Namespace: "org-acme"}, &app)
	if err != nil {
		t.Fatalf("getting app: %v", err)
	}

	return app.GetAnnotations()[annotation.AppOperatorLatestConfigMapVersion]
}

func userValues(resourceVersion string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-app-user-values",
			Namespace:       "org-acme",
			ResourceVersion: resourceVersion,
		},
	}
}

func Test_onUpdate(t *testing.T) {
	testCases := []struct {
		name            string
		oldVersion      string
		newVersion      string
		expectedVersion string
	}{
		{
			name:            "unchanged resourceVersion from a relist does not trigger",
			oldVersion:      "10",
			newVersion:      "10",
			expectedVersion: "",
		},
		{
			name:            "changed resourceVersion triggers the app",
			oldVersion:      "10",
			newVersion:      "12",
			expectedVersion: "12",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestWatcher(t)

			c.onUpdate(context.Background(), userValues(tc.oldVersion), userValues(tc.newVersion), configMapType)

			if got := latestConfigMapVersion(t, c); got != tc.expectedVersion {
				t.Fatalf("latest configmap version = %q, want %q", got, tc.expectedVersion)
			}
		})
	}
}

func Test_onAdd_recreatedConfigMap(t *testing.T) {
	c := newTestWatcher(t)

	// Deleted and created again under the same name.
	c.onDelete(context.Background(), userValues("15"), configMapType)
	c.onAdd(context.Background(), userValues("16"), configMapType)

	if got := latestConfigMapVersion(t, c); got != "16" {
		t.Fatalf("latest configmap version = %q, want %q", got, "16")
	}
}

func Test_onDelete(t *testing.T) {
	c := newTestWatcher(t)

	// The ConfigMap was updated to 15, then deleted while the watch was
	// down: the relist reports the deletion as a tombstone that still carries
	// 15, which the App already has.
	c.onUpdate(context.Background(), userValues("10"), userValues("15"), configMapType)
	if got := latestConfigMapVersion(t, c); got != "15" {
		t.Fatalf("latest configmap version after the update = %q, want %q", got, "15")
	}

	c.onDelete(context.Background(), cache.DeletedFinalStateUnknown{Key: "org-acme/my-app-user-values", Obj: userValues("15")}, configMapType)

	if got := latestConfigMapVersion(t, c); got != "15-deleted" {
		t.Fatalf("latest configmap version = %q, want %q", got, "15-deleted")
	}
}

// relistListerWatcher serves two lists of the watched ConfigMaps, in which the
// ConfigMap changed in between: a change made while the watch was down. The
// first watch expires right after the list, which makes the informer list
// again.
//
// Recent client-go informers list through a streaming watch
// (sendInitialEvents) and fall back to List, so both are served.
type relistListerWatcher struct {
	mu    sync.Mutex
	lists int
}

// nextVersion returns the resourceVersion of the ConfigMap for the next list.
func (lw *relistListerWatcher) nextVersion() string {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	lw.lists++
	if lw.lists > 1 {
		return "20"
	}

	return "10"
}

func expired() *metav1.Status {
	return &metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    http.StatusGone,
		Reason:  metav1.StatusReasonExpired,
		Message: "too old resource version",
	}
}

func (lw *relistListerWatcher) ListWithContext(_ context.Context, _ metav1.ListOptions) (runtime.Object, error) {
	version := lw.nextVersion()

	return &corev1.ConfigMapList{
		ListMeta: metav1.ListMeta{ResourceVersion: version},
		Items:    []corev1.ConfigMap{*userValues(version)},
	}, nil
}

func (lw *relistListerWatcher) WatchWithContext(_ context.Context, options metav1.ListOptions) (watch.Interface, error) {
	w := watch.NewFakeWithChanSize(3, false)

	if options.SendInitialEvents == nil || !*options.SendInitialEvents {
		// A plain watch after a List: expire at once.
		w.Error(expired())
		return w, nil
	}

	version := lw.nextVersion()
	w.Add(userValues(version))
	w.Action(watch.Bookmark, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: version,
			Annotations:     map[string]string{metav1.InitialEventsAnnotationKey: "true"},
		},
	})
	if version == "10" {
		w.Error(expired())
	}

	return w, nil
}

func (lw *relistListerWatcher) List(options metav1.ListOptions) (runtime.Object, error) {
	return lw.ListWithContext(context.Background(), options)
}

func (lw *relistListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	return lw.WatchWithContext(context.Background(), options)
}

func Test_runInformer_changeWhileWatchWasDown(t *testing.T) {
	c := newTestWatcher(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go c.runInformer(ctx, &relistListerWatcher{}, &corev1.ConfigMap{}, configMapType)

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if latestConfigMapVersion(t, c) == "20" {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("the change made while the watch was down did not trigger the app, latest configmap version = %q", latestConfigMapVersion(t, c))
}

// steadyListerWatcher serves one list and a watch that stays open without
// events.
type steadyListerWatcher struct {
	mu      sync.Mutex
	watches int
}

func (lw *steadyListerWatcher) watched() int {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	return lw.watches
}

func (lw *steadyListerWatcher) List(_ metav1.ListOptions) (runtime.Object, error) {
	return &corev1.ConfigMapList{
		ListMeta: metav1.ListMeta{ResourceVersion: "10"},
		Items:    []corev1.ConfigMap{*userValues("10")},
	}, nil
}

func (lw *steadyListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	lw.mu.Lock()
	lw.watches++
	lw.mu.Unlock()

	w := watch.NewFakeWithChanSize(2, false)
	if options.SendInitialEvents != nil && *options.SendInitialEvents {
		w.Add(userValues("10"))
		w.Action(watch.Bookmark, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "10",
				Annotations:     map[string]string{metav1.InitialEventsAnnotationKey: "true"},
			},
		})
	}

	return w, nil
}

func Test_runInformer_initialListDoesNotTrigger(t *testing.T) {
	c := newTestWatcher(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lw := &steadyListerWatcher{}
	go c.runInformer(ctx, lw, &corev1.ConfigMap{}, configMapType)

	deadline := time.Now().Add(20 * time.Second)
	for lw.watched() == 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	// Give the handlers time to process the initial list.
	time.Sleep(time.Second)

	if got := latestConfigMapVersion(t, c); got != "" {
		t.Fatalf("the initial list triggered the app, latest configmap version = %q", got)
	}
}

func Test_stripData(t *testing.T) {
	cm := userValues("10")
	cm.Data = map[string]string{"values": "big"}
	cm.ManagedFields = []metav1.ManagedFieldsEntry{{Manager: "kubectl"}}
	cm.Annotations = map[string]string{"kubectl.kubernetes.io/last-applied-configuration": `{"data":{"values":"big"}}`}

	obj, err := stripData(cm)
	if err != nil {
		t.Fatalf("stripData: %v", err)
	}

	got := obj.(*corev1.ConfigMap)
	if got.Data != nil || got.ManagedFields != nil || got.Annotations != nil || got.GetResourceVersion() != "10" || got.GetName() != "my-app-user-values" {
		t.Fatalf("stripData kept data or lost metadata: %+v", got)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"kubectl.kubernetes.io/last-applied-configuration": `{"data":{"token":"c2VjcmV0"}}`}},
		Data:       map[string][]byte{"token": []byte("secret")},
	}
	obj, err = stripData(secret)
	if err != nil {
		t.Fatalf("stripData: %v", err)
	}
	if s := obj.(*corev1.Secret); s.Data != nil || s.Annotations != nil {
		t.Fatalf("stripData kept secret data")
	}
}
