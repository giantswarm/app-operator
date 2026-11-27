package appvalue

import (
	"context"

	"github.com/giantswarm/k8smetadata/pkg/label"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// watchConfigMap triggers an update of the Apps that depend on a ConfigMap
// labelled for watching whenever that ConfigMap changes.
func (c *AppValueWatcher) watchConfigMap(ctx context.Context) {
	lw := &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = label.AppOperatorWatching
			return c.k8sClient.K8sClient().CoreV1().ConfigMaps(metav1.NamespaceAll).List(ctx, options)
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = label.AppOperatorWatching
			return c.k8sClient.K8sClient().CoreV1().ConfigMaps(metav1.NamespaceAll).Watch(ctx, options)
		},
	}

	c.runInformer(ctx, lw, &corev1.ConfigMap{}, configMapType)
}
