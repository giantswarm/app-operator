//go:build k8srequired
// +build k8srequired

package configmap

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/v6/integration/key"
)

// TestWatchingConfigMap tests app CRs are updated when wired configmaps are updated
//
// - Create user configmap, catalog configmap
//
// - Create app CR and wiring user configmap and catalog
//
//   - Update user configmap and check the latest resource version is set on the annotation
//     of app CR.
//
//   - Update appcatalog configmap and check the latest resource version is set on the annotation
//     of app CR.
//
// - Delete app CR and check the watching label is deleted.
func TestWatchingConfigMap(t *testing.T) {
	ctx := context.Background()

	var cr v1alpha1.App
	var err error

	{
		config.Logger.Debugf(ctx, "creating configmap %#q in namespace %#q", key.CatalogConfigMapName(), key.GiantSwarmNamespace())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.CatalogConfigMapName(),
				Namespace: key.GiantSwarmNamespace(),
			},
			Data: map[string]string{
				"values": "",
			},
		}

		_, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.GiantSwarmNamespace()).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "created configmap %#q in namespace %#q", key.CatalogConfigMapName(), key.GiantSwarmNamespace())
	}

	{
		config.Logger.Debugf(ctx, "creating %#q appcatalog cr", key.DefaultCatalogName())

		catalogCR := &v1alpha1.Catalog{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.DefaultCatalogName(),
				Namespace: key.GiantSwarmNamespace(),
				Labels: map[string]string{
					label.AppOperatorVersion: key.UniqueAppVersion(),
				},
			},
			Spec: v1alpha1.CatalogSpec{
				Config: &v1alpha1.CatalogSpecConfig{
					ConfigMap: &v1alpha1.CatalogSpecConfigConfigMap{
						Name:      key.CatalogConfigMapName(),
						Namespace: key.GiantSwarmNamespace(),
					},
				},
				Description: key.DefaultCatalogName(),
				Title:       key.DefaultCatalogName(),
				Repositories: []v1alpha1.CatalogSpecRepository{
					{
						Type: "helm",
						URL:  key.DefaultCatalogStorageURL(),
					},
				},
				Storage: v1alpha1.CatalogSpecStorage{
					Type: "helm",
					URL:  key.DefaultCatalogStorageURL(),
				},
			},
		}
		err = config.K8sClients.CtrlClient().Create(ctx, catalogCR)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "created %#q catalog cr in namespace %#q", key.DefaultCatalogName(), key.GiantSwarmNamespace())
	}

	{
		config.Logger.Debugf(ctx, "creating configmap %#q in namespace %#q", key.UserConfigMapName(), key.GiantSwarmNamespace())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.UserConfigMapName(),
				Namespace: key.GiantSwarmNamespace(),
			},
			Data: map[string]string{
				"values": "",
			},
		}

		_, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.GiantSwarmNamespace()).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "created configmap %#q in namespace %#q", key.UserConfigMapName(), key.GiantSwarmNamespace())
	}

	{
		config.Logger.Debugf(ctx, "creating %#q app cr", key.TestAppName())

		appCR := &v1alpha1.App{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.TestAppName(),
				Namespace: key.GiantSwarmNamespace(),
				Labels: map[string]string{
					label.AppOperatorVersion: key.UniqueAppVersion(),
				},
			},
			Spec: v1alpha1.AppSpec{
				Catalog: key.DefaultCatalogName(),
				KubeConfig: v1alpha1.AppSpecKubeConfig{
					InCluster: true,
				},
				Name:      key.TestAppName(),
				Namespace: key.GiantSwarmNamespace(),
				UserConfig: v1alpha1.AppSpecUserConfig{
					ConfigMap: v1alpha1.AppSpecUserConfigConfigMap{
						Name:      key.UserConfigMapName(),
						Namespace: key.GiantSwarmNamespace(),
					},
				},
				Version: "0.1.0",
			},
		}

		err = config.K8sClients.CtrlClient().Create(ctx, appCR)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "creating %#q app cr", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "waiting until user configmap is labelled")

		o := func() error {
			cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.GiantSwarmNamespace()).Get(ctx, key.UserConfigMapName(), metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			if _, ok := cm.GetLabels()[label.AppOperatorWatching]; !ok {
				return microerror.Maskf(notFoundError, fmt.Sprintf("%#q label not found", label.AppOperatorWatching))
			}

			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.Errorf(ctx, err, "failed to get a label; retrying in %d", t)
		}

		b := backoff.NewMaxRetries(5, backoff.ShortMaxInterval)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "waited until user configmap was labelled")
	}

	{
		config.Logger.Debugf(ctx, "waiting until appcatalog configmap is labelled")

		o := func() error {
			cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.GiantSwarmNamespace()).Get(ctx, key.CatalogConfigMapName(), metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			if _, ok := cm.GetLabels()[label.AppOperatorWatching]; !ok {
				return microerror.Maskf(notFoundError, fmt.Sprintf("%#q label not found", label.AppOperatorWatching))
			}

			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.Errorf(ctx, err, "failed to get a label; retrying in %d", t)
		}

		b := backoff.NewMaxRetries(5, backoff.ShortMaxInterval)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "waited until appcatalog configmap was labelled")
	}

	var updatedResourceVersion string
	{
		config.Logger.Debugf(ctx, "updating values in configmap %#q in namespace %#q", key.UserConfigMapName(), key.GiantSwarmNamespace())

		cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.GiantSwarmNamespace()).Get(ctx, key.UserConfigMapName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		cm.Data["values"] = "test: userconfigmap"
		updatedCM, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.GiantSwarmNamespace()).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		updatedResourceVersion = updatedCM.GetResourceVersion()

		config.Logger.Debugf(ctx, "updated values in configmap %#q in namespace %#q", key.UserConfigMapName(), key.GiantSwarmNamespace())
	}

	versionAnnotation := annotation.AppOperatorLatestConfigMapVersion

	{
		config.Logger.Debugf(ctx, "waiting until app CR is annotated with user configmap's resourceVersion")

		o := func() error {
			err = config.K8sClients.CtrlClient().Get(
				ctx,
				types.NamespacedName{Name: key.TestAppName(), Namespace: key.GiantSwarmNamespace()},
				&cr,
			)
			if err != nil {
				return microerror.Mask(err)
			}

			if v, ok := cr.GetAnnotations()[versionAnnotation]; !ok {
				return microerror.Maskf(notFoundError, fmt.Sprintf("%#q annotation not found", versionAnnotation))
			} else if v != updatedResourceVersion {
				return microerror.Maskf(testError, fmt.Sprintf("expect annotation equal to %#q but %#q", updatedResourceVersion, v))
			}

			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.Errorf(ctx, err, "failed to get an annotation; retrying in %d", t)
		}

		b := backoff.NewMaxRetries(5, backoff.ShortMaxInterval)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "waited until app CR is annotated with user configmap's resourceVersion")
	}

	{
		config.Logger.Debugf(ctx, "editing configmap %#q in namespace %#q", key.CatalogConfigMapName(), key.GiantSwarmNamespace())

		cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.GiantSwarmNamespace()).Get(ctx, key.CatalogConfigMapName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		cm.Data["values"] = "test: appcatalogConfigmap"
		updatedCM, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.GiantSwarmNamespace()).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		updatedResourceVersion = updatedCM.GetResourceVersion()

		config.Logger.Debugf(ctx, "edited configmap %#q in namespace %#q", key.UserConfigMapName(), key.GiantSwarmNamespace())
	}

	{
		config.Logger.Debugf(ctx, "waiting until app CR annotate by appcatalog configmap's resourceVersion")

		o := func() error {
			err = config.K8sClients.CtrlClient().Get(
				ctx,
				types.NamespacedName{Name: key.TestAppName(), Namespace: key.GiantSwarmNamespace()},
				&cr,
			)
			if err != nil {
				return microerror.Mask(err)
			}

			if v, ok := cr.GetAnnotations()[versionAnnotation]; !ok {
				return microerror.Maskf(notFoundError, fmt.Sprintf("%#q annotation not found", versionAnnotation))
			} else if v != updatedResourceVersion {
				return microerror.Maskf(testError, fmt.Sprintf("expect annotation equal to %#q but %#q", updatedResourceVersion, v))
			}

			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.Errorf(ctx, err, "failed to get an annotation; retrying in %s", t)
		}

		b := backoff.NewMaxRetries(5, backoff.ShortMaxInterval)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "waited until app CR annotate by appcatalog configmap's resourceVersion")
	}

	{
		config.Logger.Debugf(ctx, "deleting app CR %#q", key.TestAppName())

		err = config.K8sClients.CtrlClient().Delete(ctx, &cr)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "deleted app CR %#q", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "waiting until watching label is deleted")

		o := func() error {
			cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.GiantSwarmNamespace()).Get(ctx, key.UserConfigMapName(), metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			if _, ok := cm.GetLabels()[label.AppOperatorWatching]; ok {
				return microerror.Maskf(testError, fmt.Sprintf("%#q label still found", label.AppOperatorWatching))
			}

			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.Errorf(ctx, err, "still getting label; retrying in %s", t)
		}

		b := backoff.NewMaxRetries(5, backoff.ShortMaxInterval)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "waited until watching label was deleted")
	}

}
