// +build k8srequired

package appcatalogentry

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v2/integration/key"
	"github.com/giantswarm/app-operator/v2/pkg/project"
)

// TestAppCatalogEntry tests appcatalogentry CRs are generated for the
// giantswarm catalog.
//
// Create giantswarm appcatalog CR to trigger creation of appcatalogentry CRs.
// Get a single CR and check values are correct.
//
// Delete giantswarm appcatalog CR to trigger deletion of appcatalogentry CRs.
// Check all appcatalogentry CRs are deleted.
//
func TestAppCatalogEntry(t *testing.T) {
	ctx := context.Background()

	var err error

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating %#q appcatalog cr", key.StableCatalogName()))

		appCatalogCR := &v1alpha1.AppCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: key.StableCatalogName(),
				Labels: map[string]string{
					label.AppOperatorVersion:   project.Version(),
					label.CatalogType:       "stable",
					label.CatalogVisibility: "public",
				},
			},
			Spec: v1alpha1.AppCatalogSpec{
				Description: key.StableCatalogName(),
				Title:       key.StableCatalogName(),
				Storage: v1alpha1.AppCatalogSpecStorage{
					Type: "helm",
					URL:  key.StableCatalogStorageURL(),
				},
			},
		}
		_, err = config.K8sClients.G8sClient().ApplicationV1alpha1().AppCatalogs().Create(ctx, appCatalogCR, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created %#q appcatalog cr", key.StableCatalogName()))
	}

	var entryCR *v1alpha1.AppCatalogEntry

	{
		o := func() error {
			entryCR, err = config.K8sClients.G8sClient().ApplicationV1alpha1().AppCatalogEntries(metav1.NamespaceDefault).Get(ctx, key.AppCatalogEntryName(), metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("failed to get appcatalogentry CR with name %#q: retrying in %s", key.AppCatalogEntryName(), t), "stack", fmt.Sprintf("%v", err))
		}

		b := backoff.NewConstant(5*time.Minute, 15*time.Second)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	{
		expectedLabels := map[string]string{
			pkglabel.AppKubernetesName: "prometheus-operator-app",
			pkglabel.CatalogName:       key.StableCatalogName(),
			pkglabel.CatalogType:       "stable",
			pkglabel.Latest:            "false",
			label.ManagedBy:            "app-operator-unique",
		}

		if !reflect.DeepEqual(entryCR.Labels, expectedLabels) {
			t.Fatalf("want matching labels \n %s", cmp.Diff(entryCR.Labels, expectedLabels))
		}

		expectedEntrySpec := v1alpha1.AppCatalogEntrySpec{
			AppName:    "prometheus-operator-app",
			AppVersion: "0.38.1",
			Catalog: v1alpha1.AppCatalogEntrySpecCatalog{
				Name:      key.StableCatalogName(),
				Namespace: "",
			},
			Chart: v1alpha1.AppCatalogEntrySpecChart{
				Home: "https://github.com/giantswarm/prometheus-operator-app",
				Icon: "https://raw.githubusercontent.com/prometheus/prometheus.github.io/master/assets/prometheus_logo-cb55bb5c346.png",
			},
			DateCreated: nil,
			DateUpdated: nil,
			Version:     "0.3.4",
		}

		// Clear dates for comparison.
		entryCR.Spec.DateCreated = nil
		entryCR.Spec.DateUpdated = nil

		if !reflect.DeepEqual(entryCR.Spec, expectedEntrySpec) {
			t.Fatalf("want matching spec \n %s", cmp.Diff(entryCR.Spec, expectedEntrySpec))
		}
	}

	{
		err = config.K8sClients.G8sClient().ApplicationV1alpha1().AppCatalogs().Delete(ctx, key.StableCatalogName(), metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		o := func() error {
			lo := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.ManagedBy, project.Name(), pkglabel.CatalogName, key.StableCatalogName()),
			}
			entryCRs, err := config.K8sClients.G8sClient().ApplicationV1alpha1().AppCatalogEntries(metav1.NamespaceDefault).List(ctx, lo)
			if err != nil {
				return microerror.Mask(err)
			}
			if len(entryCRs.Items) > 0 {
				return microerror.Maskf(testError, "expected 0 appcatalogentries got %d", len(entryCRs.Items))
			}

			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("appcatalogentry CRs still exist: retrying in %s", t), "stack", fmt.Sprintf("%v", err))
		}

		b := backoff.NewMaxRetries(10, 15*time.Second)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}
}
