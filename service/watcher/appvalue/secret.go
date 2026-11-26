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

// watchSecret triggers an update of the Apps that depend on a Secret labelled
// for watching whenever that Secret changes. Only the secret namespace is
// watched.
func (c *AppValueWatcher) watchSecret(ctx context.Context) {
	lw := &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = label.AppOperatorWatching
			return c.k8sClient.K8sClient().CoreV1().Secrets(c.secretNamespace).List(ctx, options)
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = label.AppOperatorWatching
			return c.k8sClient.K8sClient().CoreV1().Secrets(c.secretNamespace).Watch(ctx, options)
		},
	}

	c.runInformer(ctx, lw, &corev1.Secret{}, secretType)
}
