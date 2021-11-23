//go:build k8srequired
// +build k8srequired

package configmap

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/app/v4/pkg/annotation"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v3/integration/key"
	pkglabel "github.com/giantswarm/app-operator/v3/pkg/label"
)

// TestWatchingConfigMap tests app CRs are updated when wired configmaps are updated
//
// - Create user configmap, appcatalog configmap
//
// - Create app CR and wiring user configmap and appcatalog
//
// - Update user configmap and check the latest resource version is set on the annotation
//   of app CR.
//
// - Update appcatalog onfigmap and check the latest resource version is set on the annotation
//   of app CR.
//
//
// - Delete app CR and check the watching label is deleted.
//
func TestWatchingConfigMap(t *testing.T) {
	ctx := context.Background()

	var err error

	{
		config.Logger.Debugf(ctx, "creating configmap %#q in namespace %#q", key.AppCatalogConfigMapName(), key.Namespace())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.AppCatalogConfigMapName(),
				Namespace: key.Namespace(),
			},
			Data: map[string]string{
				"values": "",
			},
		}

		_, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.Namespace()).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "created configmap %#q in namespace %#q", key.AppCatalogConfigMapName(), key.Namespace())
	}

	{
		config.Logger.Debugf(ctx, "creating %#q appcatalog cr", key.DefaultCatalogName())

		appCatalogCR := &v1alpha1.AppCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: key.DefaultCatalogName(),
				Labels: map[string]string{
					label.AppOperatorVersion: key.UniqueAppVersion(),
				},
			},
			Spec: v1alpha1.AppCatalogSpec{
				Config: v1alpha1.AppCatalogSpecConfig{
					ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
						Name:      key.AppCatalogConfigMapName(),
						Namespace: key.Namespace(),
					},
				},
				Description: key.DefaultCatalogName(),
				Title:       key.DefaultCatalogName(),
				Storage: v1alpha1.AppCatalogSpecStorage{
					Type: "helm",
					URL:  key.DefaultCatalogStorageURL(),
				},
			},
		}
		_, err = config.K8sClients.G8sClient().ApplicationV1alpha1().AppCatalogs().Create(ctx, appCatalogCR, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "created %#q appcatalog cr", key.DefaultCatalogName())
	}

	{
		config.Logger.Debugf(ctx, "creating configmap %#q in namespace %#q", key.UserConfigMapName(), key.Namespace())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.UserConfigMapName(),
				Namespace: key.Namespace(),
			},
			Data: map[string]string{
				"values": "",
			},
		}

		_, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.Namespace()).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "created configmap %#q in namespace %#q", key.UserConfigMapName(), key.Namespace())
	}

	{
		config.Logger.Debugf(ctx, "creating %#q app cr", key.TestAppName())

		appCR := &v1alpha1.App{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.TestAppName(),
				Namespace: key.Namespace(),
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
				Namespace: key.Namespace(),
				UserConfig: v1alpha1.AppSpecUserConfig{
					ConfigMap: v1alpha1.AppSpecUserConfigConfigMap{
						Name:      key.UserConfigMapName(),
						Namespace: key.Namespace(),
					},
				},
				Version: "0.1.0",
			},
		}

		_, err = config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(key.Namespace()).Create(ctx, appCR, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "creating %#q app cr", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "waiting until user configmap is labelled")

		o := func() error {
			cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.Namespace()).Get(ctx, key.UserConfigMapName(), metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			if _, ok := cm.GetLabels()[pkglabel.Watching]; !ok {
				return microerror.Maskf(notFoundError, fmt.Sprintf("%#q label not found", pkglabel.Watching))
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
			cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.Namespace()).Get(ctx, key.AppCatalogConfigMapName(), metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			if _, ok := cm.GetLabels()[pkglabel.Watching]; !ok {
				return microerror.Maskf(notFoundError, fmt.Sprintf("%#q label not found", pkglabel.Watching))
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
		config.Logger.Debugf(ctx, "updating values in configmap %#q in namespace %#q", key.UserConfigMapName(), key.Namespace())

		cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.Namespace()).Get(ctx, key.UserConfigMapName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		cm.Data["values"] = "test: userconfigmap"
		updatedCM, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.Namespace()).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		updatedResourceVersion = updatedCM.GetResourceVersion()

		config.Logger.Debugf(ctx, "updated values in configmap %#q in namespace %#q", key.UserConfigMapName(), key.Namespace())
	}

	versionAnnotation := fmt.Sprintf("%s/%s", annotation.AppOperatorPrefix, annotation.LatestConfigMapVersion)

	{
		config.Logger.Debugf(ctx, "waiting until app CR is annotated with user configmap's resourceVersion")

		o := func() error {
			cr, err := config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(key.Namespace()).Get(ctx, key.TestAppName(), metav1.GetOptions{})
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
		config.Logger.Debugf(ctx, "editing configmap %#q in namespace %#q", key.AppCatalogConfigMapName(), key.Namespace())

		cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.Namespace()).Get(ctx, key.AppCatalogConfigMapName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		cm.Data["values"] = "test: appcatalogConfigmap"
		updatedCM, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.Namespace()).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		updatedResourceVersion = updatedCM.GetResourceVersion()

		config.Logger.Debugf(ctx, "edited configmap %#q in namespace %#q", key.UserConfigMapName(), key.Namespace())
	}

	{
		config.Logger.Debugf(ctx, "waiting until app CR annotate by appcatalog configmap's resourceVersion")

		o := func() error {
			cr, err := config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(key.Namespace()).Get(ctx, key.TestAppName(), metav1.GetOptions{})
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

		config.Logger.Debugf(ctx, "waited until app CR annotate by appcatalog configmap's resourceVersion")
	}

	{
		config.Logger.Debugf(ctx, "deleting app CR %#q", key.TestAppName())

		err = config.K8sClients.G8sClient().ApplicationV1alpha1().Apps(key.Namespace()).Delete(ctx, key.TestAppName(), metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "deleted app CR %#q", key.TestAppName())
	}

	{
		config.Logger.Debugf(ctx, "waiting until watching label is deleted")

		o := func() error {
			cm, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(key.Namespace()).Get(ctx, key.UserConfigMapName(), metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			if _, ok := cm.GetLabels()[pkglabel.Watching]; ok {
				return microerror.Maskf(testError, fmt.Sprintf("%#q label still found", pkglabel.Watching))
			}

			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.Errorf(ctx, err, "still getting label; retrying in %d", t)
		}

		b := backoff.NewMaxRetries(5, backoff.ShortMaxInterval)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "waited until watching label was deleted")
	}

}
