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
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v3/integration/key"
	pkglabel "github.com/giantswarm/app-operator/v3/pkg/label"
	"github.com/giantswarm/app-operator/v3/pkg/project"
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
		config.Logger.Debugf(ctx, "creating %#q appcatalog cr", key.StableCatalogName())

		appCatalogCR := &v1alpha1.AppCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: key.StableCatalogName(),
				Labels: map[string]string{
					label.AppOperatorVersion: project.Version(),
					label.CatalogType:        "stable",
					label.CatalogVisibility:  "public",
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

		config.Logger.Debugf(ctx, "created %#q appcatalog cr", key.StableCatalogName())
	}

	var latestEntry appcatalog.Entry
	{
		latestEntry, err = appcatalog.GetLatestEntry(ctx, key.StableCatalogStorageURL(), "prometheus-operator-app", "")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	var entryCR *v1alpha1.AppCatalogEntry
	{
		appCatalogEntryName := fmt.Sprintf("%s-%s-%s", key.GiantSwarmNamespace(), latestEntry.Name, latestEntry.Version)

		o := func() error {
			entryCR, err = config.K8sClients.G8sClient().ApplicationV1alpha1().AppCatalogEntries(metav1.NamespaceDefault).Get(ctx, appCatalogEntryName, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.Errorf(ctx, err, "failed to get appcatalogentry CR with name %#q: retrying in %s", appCatalogEntryName, t)
		}

		b := backoff.NewConstant(5*time.Minute, 15*time.Second)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	{
		expectedLabels := map[string]string{
			label.AppKubernetesName: "prometheus-operator-app",
			label.CatalogName:       key.StableCatalogName(),
			label.CatalogType:       "stable",
			pkglabel.Latest:         "true",
			label.ManagedBy:         "app-operator-unique",
		}

		if !reflect.DeepEqual(entryCR.Labels, expectedLabels) {
			t.Fatalf("want matching labels \n %s", cmp.Diff(entryCR.Labels, expectedLabels))
		}

		expectedEntrySpec := v1alpha1.AppCatalogEntrySpec{
			AppName:    latestEntry.Name,
			AppVersion: latestEntry.AppVersion,
			Catalog: v1alpha1.AppCatalogEntrySpecCatalog{
				Name:      key.StableCatalogName(),
				Namespace: "",
			},
			Chart: v1alpha1.AppCatalogEntrySpecChart{
				APIVersion: "v1",
				Home:       latestEntry.Home,
				Icon:       latestEntry.Icon,
			},
			DateCreated: nil,
			DateUpdated: nil,
			Version:     latestEntry.Version,
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
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.ManagedBy, project.Name(), label.CatalogName, key.StableCatalogName()),
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
			config.Logger.Errorf(ctx, err, "appcatalogentry CRs still exist: retrying in %s", t)
		}

		b := backoff.NewMaxRetries(10, 15*time.Second)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}
}
