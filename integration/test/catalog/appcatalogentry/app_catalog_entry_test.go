//go:build k8srequired
// +build k8srequired

package appcatalogentry

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-operator/v7/integration/key"
	pkglabel "github.com/giantswarm/app-operator/v7/pkg/label"
	"github.com/giantswarm/app-operator/v7/pkg/project"
)

// TestAppCatalogEntry tests appcatalogentry CRs are generated for the
// giantswarm catalog.
//
// Create giantswarm catalog CR to trigger creation of appcatalogentry CRs.
// Get a single CR and check values are correct.
//
// Delete giantswarm catalog CR to trigger deletion of appcatalogentry CRs.
// Check all appcatalogentry CRs are deleted.
func TestAppCatalogEntry(t *testing.T) {
	ctx := context.Background()

	var catalogCR v1alpha1.Catalog
	var err error

	{
		config.Logger.Debugf(ctx, "creating %#q catalog cr", key.StableCatalogName())

		catalogCR = v1alpha1.Catalog{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.StableCatalogName(),
				Namespace: metav1.NamespaceDefault,
				Labels: map[string]string{
					label.AppOperatorVersion: project.Version(),
					label.CatalogType:        "stable",
					label.CatalogVisibility:  "public",
				},
			},
			Spec: v1alpha1.CatalogSpec{
				Description: key.StableCatalogName(),
				Title:       key.StableCatalogName(),
				Storage: v1alpha1.CatalogSpecStorage{
					Type: "helm",
					URL:  key.StableCatalogStorageURL(),
				},
				Repositories: []v1alpha1.CatalogSpecRepository{
					{
						Type: "helm",
						URL:  key.StableCatalogStorageURL(),
					},
				},
			},
		}
		err = config.K8sClients.CtrlClient().Create(ctx, &catalogCR)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "created %#q catalog cr", key.StableCatalogName())
	}

	var latestEntry appcatalog.Entry
	{
		latestEntry, err = appcatalog.GetLatestEntry(ctx, key.StableCatalogStorageURL(), "prometheus-operator-app", "")
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	var entryCR v1alpha1.AppCatalogEntry
	{
		appCatalogEntryName := fmt.Sprintf("%s-%s-%s", key.GiantSwarmNamespace(), latestEntry.Name, latestEntry.Version)

		o := func() error {
			err = config.K8sClients.CtrlClient().Get(
				ctx,
				types.NamespacedName{Name: appCatalogEntryName, Namespace: metav1.NamespaceDefault},
				&entryCR,
			)
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
		// Set latest label to false to stop the test from flapping. This is
		// because the latest release may not be the latest according to semver.
		entryCR.Labels[pkglabel.Latest] = "false"

		expectedLabels := map[string]string{
			label.AppKubernetesName:    "prometheus-operator-app",
			label.AppKubernetesVersion: latestEntry.Version,
			label.CatalogName:          key.StableCatalogName(),
			label.CatalogType:          "stable",
			pkglabel.Latest:            "false",
			label.ManagedBy:            "app-operator-unique",
		}

		if !reflect.DeepEqual(entryCR.Labels, expectedLabels) {
			t.Fatalf("want matching labels \n %s", cmp.Diff(entryCR.Labels, expectedLabels))
		}

		expectedEntrySpec := v1alpha1.AppCatalogEntrySpec{
			AppName:    latestEntry.Name,
			AppVersion: latestEntry.AppVersion,
			Catalog: v1alpha1.AppCatalogEntrySpecCatalog{
				Name:      key.StableCatalogName(),
				Namespace: metav1.NamespaceDefault,
			},
			Chart: v1alpha1.AppCatalogEntrySpecChart{
				APIVersion:  "v2",
				Description: latestEntry.Description,
				Home:        latestEntry.Home,
				Icon:        latestEntry.Icon,
				Keywords:    latestEntry.Keywords,
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
		err = config.K8sClients.CtrlClient().Delete(ctx, &catalogCR)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		entryCRs := v1alpha1.AppCatalogEntryList{}

		catalogLabels, err := labels.Parse(fmt.Sprintf("%s=%s,%s=%s", label.ManagedBy, project.Name(), label.CatalogName, key.StableCatalogName()))
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		o := func() error {
			err = config.K8sClients.CtrlClient().List(ctx, &entryCRs, &client.ListOptions{LabelSelector: catalogLabels})
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
		err = backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}
}
