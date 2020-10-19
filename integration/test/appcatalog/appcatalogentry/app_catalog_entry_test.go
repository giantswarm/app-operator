// +build k8srequired

package appcatalogentry

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/crd"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v2/integration/key"
	pkglabel "github.com/giantswarm/app-operator/v2/pkg/label"
	"github.com/giantswarm/app-operator/v2/pkg/project"
)

// TestAppCatalogEntry tests appcatalogentry CRs are generated for the
// giantswarm catalog.
//
func TestAppCatalogEntry(t *testing.T) {
	ctx := context.Background()

	var err error

	{
		crdName := "AppCatalogEntry"
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring %#q CRD exists", crdName))

		err = config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("application.giantswarm.io", crdName), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured %#q CRD exists", crdName))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating %#q appcatalog cr", key.StableCatalogName()))

		appCatalogCR := &v1alpha1.AppCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: key.DefaultCatalogName(),
				Labels: map[string]string{
					label.AppOperatorVersion:   project.Version(),
					pkglabel.CatalogType:       "stable",
					pkglabel.CatalogVisibility: "public",
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
		expectedLabels := map[string]string{}

		if !reflect.DeepEqual(entryCR.Labels, expectedLabels) {
			t.Fatalf("want matching labels \n %s", cmp.Diff(entryCR.Labels, expectedLabels))
		}
	}
}
