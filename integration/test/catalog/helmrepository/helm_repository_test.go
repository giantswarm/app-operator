//go:build k8srequired
// +build k8srequired

package helmrepository

import (
	"context"
	"reflect"
	"testing"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-operator/v6/integration/key"
	operatorkey "github.com/giantswarm/app-operator/v6/pkg/key"
	"github.com/giantswarm/app-operator/v6/pkg/project"
)

const (
	catalog = "giantswarm"
)

// TestCatalogToHelmRepository tests Catalog CR translation to HelmRepository CRs.
//
// It creates `giantswarm` Catalog CR to trigger creation of HelmRepository CRs out
// of it.
//
// It then updates `giantswarm` Catalog CR by changing one of its repositories URL,
// to trigger update of the HelmRepository CRs set.
//
// At the end it deletes `giantswarm` Catalog CR to trigger deletion of HelmRepository CRs,
// created previously.
func TestCatalogToHelmRepository(t *testing.T) {
	ctx := context.Background()

	var catalog v1alpha1.Catalog
	var err error

	// Stage 1: create a Cataog CR and watch for expected HelmRepository CRs
	// being created.
	{
		catalog = v1alpha1.Catalog{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "giantswarm",
				Namespace: "giantswarm", //metav1.NamespaceDefault,
				Labels: map[string]string{
					label.AppOperatorVersion: project.Version(),
					label.CatalogType:        "stable",
					label.CatalogVisibility:  "public",
				},
			},
			Spec: v1alpha1.CatalogSpec{
				Description: "Giantswarm Catalog",
				Title:       "Giantswarm Catalog",
				Storage: v1alpha1.CatalogSpecStorage{
					Type: "helm",
					URL:  key.TestCatalogStorageHelmURL(),
				},
				Repositories: []v1alpha1.CatalogSpecRepository{
					{
						Type: "helm",
						URL:  key.TestCatalogStorageHelmURL(),
					},
					{
						Type: "oci",
						URL:  key.StableCatalogStorageOciURL(),
					},
				},
			},
		}

		config.Logger.Debugf(ctx, "creating %#q Catalog CR", catalog.Name)

		err = config.K8sClients.CtrlClient().Create(ctx, &catalog)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "created %#q Catalog CR", catalog.Name)
	}

	var expectedHRs []*sourcev1.HelmRepository
	{
		expectedHRs, err = getExpectedHRs(&catalog)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		for _, expected := range expectedHRs {

			config.Logger.Debugf(ctx, "verifying %#q HelmRepository CR existence and configuration", expected.Name)

			current := &sourcev1.HelmRepository{}
			{
				o := func() error {
					err = config.K8sClients.CtrlClient().Get(
						ctx,
						types.NamespacedName{Name: expected.Name, Namespace: expected.Namespace},
						current,
					)
					if err != nil {
						return microerror.Mask(err)
					}

					return nil
				}

				n := func(err error, t time.Duration) {
					config.Logger.Errorf(ctx, err, "failed to get HelmRepository CR with name %#q: retrying in %s", expected.Name, t)
				}

				b := backoff.NewConstant(5*time.Minute, 15*time.Second)
				err := backoff.RetryNotify(o, b, n)
				if err != nil {
					t.Fatalf("expected %#v got %#v", nil, err)
				}
			}

			if !reflect.DeepEqual(current.Labels, expected.Labels) {
				t.Fatalf("want matching labels for %s \n %s", expected.Name, cmp.Diff(current.Labels, expected.Labels))
			}

			if !reflect.DeepEqual(current.Spec, expected.Spec) {
				t.Fatalf("want matching spec for %s \n %s", expected.Name, cmp.Diff(current.Spec, expected.Spec))
			}

			config.Logger.Debugf(ctx, "verified %#q HelmRepository CR existence and configuration", expected.Name)
		}
	}

	var currentCatalog v1alpha1.Catalog
	{
		config.Logger.Debugf(ctx, "verifying %#q Catalog CR status", catalog.Name)

		err = config.K8sClients.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: catalog.Name, Namespace: catalog.Namespace},
			&currentCatalog,
		)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		if currentCatalog.Status.HelmRepositoryList == nil {
			t.Fatalf("expected list got %#v", nil)
		}

	Stage1Entry:
		for _, entry := range expectedHRs {
			config.Logger.Debugf(
				ctx,
				"verifying %#q HelmRepository CR, from %#q namespace, presence in Catalog CR status",
				entry.Name,
				entry.Namespace,
			)

			for _, hr := range currentCatalog.Status.HelmRepositoryList.Entries {
				if entry.Name != hr.Name {
					continue
				}

				if entry.Namespace != hr.Namespace {
					continue
				}

				config.Logger.Debugf(
					ctx,
					"%#q HelmRepository CR, from %#q namespace, present in Catalog CR status",
					entry.Name,
					entry.Namespace,
				)

				continue Stage1Entry
			}

			t.Fatalf("%#q HelmRepository in %#q namespace not found in Catalog CR status", entry.Name, entry.Namespace)
		}
	}

	// Stage 2: patch the Catalog CR changing test repository URL to a stable URL, and observe
	// HelmRepository CRs being both created and deleted.
	{
		expectedHRToDelete := expectedHRs[0]

		newCatalogSpec := v1alpha1.CatalogSpec{
			Description: "Giantswarm Catalog",
			Title:       "Giantswarm Catalog",
			Storage: v1alpha1.CatalogSpecStorage{
				Type: "helm",
				URL:  key.StableCatalogStorageHelmURL(),
			},
			Repositories: []v1alpha1.CatalogSpecRepository{
				{
					Type: "helm",
					URL:  key.StableCatalogStorageHelmURL(),
				},
				{
					Type: "oci",
					URL:  key.StableCatalogStorageOciURL(),
				},
			},
		}

		updatedCatalog := currentCatalog.DeepCopy()
		updatedCatalog.Spec = newCatalogSpec

		config.Logger.Debugf(ctx, "patching %#q Catalog CR", catalog.Name)

		err = config.K8sClients.CtrlClient().Patch(ctx, updatedCatalog, client.MergeFrom(&currentCatalog))
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "patched %#q Catalog CR", catalog.Name)

		expectedHRs, err = getExpectedHRs(updatedCatalog)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		for _, expected := range expectedHRs {

			config.Logger.Debugf(ctx, "verifying %#q HelmRepository CR existence and configuration", expected.Name)

			current := &sourcev1.HelmRepository{}
			{
				o := func() error {
					err = config.K8sClients.CtrlClient().Get(
						ctx,
						types.NamespacedName{Name: expected.Name, Namespace: expected.Namespace},
						current,
					)
					if err != nil {
						return microerror.Mask(err)
					}

					return nil
				}

				n := func(err error, t time.Duration) {
					config.Logger.Errorf(ctx, err, "failed to get HelmRepository CR with name %#q: retrying in %s", expected.Name, t)
				}

				b := backoff.NewConstant(5*time.Minute, 15*time.Second)
				err := backoff.RetryNotify(o, b, n)
				if err != nil {
					t.Fatalf("expected %#v got %#v", nil, err)
				}
			}

			if !reflect.DeepEqual(current.Labels, expected.Labels) {
				t.Fatalf("want matching labels for %s \n %s", expected.Name, cmp.Diff(current.Labels, expected.Labels))
			}

			if !reflect.DeepEqual(current.Spec, expected.Spec) {
				t.Fatalf("want matching spec for %s \n %s", expected.Name, cmp.Diff(current.Spec, expected.Spec))
			}

			config.Logger.Debugf(ctx, "verified %#q HelmRepository CR existence and configuration", expected.Name)
		}

		{
			config.Logger.Debugf(ctx, "verifying %#q Catalog CR status", catalog.Name)

			err = config.K8sClients.CtrlClient().Get(
				ctx,
				types.NamespacedName{Name: catalog.Name, Namespace: catalog.Namespace},
				&currentCatalog,
			)
			if err != nil {
				t.Fatalf("expected %#v got %#v", nil, err)
			}

			if currentCatalog.Status.HelmRepositoryList == nil {
				t.Fatalf("expected list got %#v", nil)
			}

		Stage2Entry:
			for _, entry := range expectedHRs {
				config.Logger.Debugf(
					ctx,
					"verifying %#q HelmRepository CR, from %#q namespace, presence in Catalog CR status",
					entry.Name,
					entry.Namespace,
				)

				for _, hr := range currentCatalog.Status.HelmRepositoryList.Entries {
					if entry.Name != hr.Name {
						continue
					}

					if entry.Namespace != hr.Namespace {
						continue
					}

					config.Logger.Debugf(
						ctx,
						"%#q HelmRepository CR, from %#q namespace, present in Catalog CR status",
						entry.Name,
						entry.Namespace,
					)

					continue Stage2Entry
				}

				t.Fatalf("%#q HelmRepository, from %#q namespace, not found in Catalog CR status", entry.Name, entry.Namespace)
			}
		}

		config.Logger.Debugf(
			ctx,
			"verifying outdated %#q HelmRepository CR, from %#q namespace, is gone",
			expectedHRToDelete.Name,
			expectedHRToDelete.Namespace,
		)

		err = config.K8sClients.CtrlClient().Get(
			ctx,
			types.NamespacedName{Name: expectedHRToDelete.Name, Namespace: expectedHRToDelete.Namespace},
			&sourcev1.HelmRepository{},
		)
		if !apierrors.IsNotFound(err) {
			t.Fatalf("expected NotFoundError got %#v", err)
		}

		config.Logger.Debugf(
			ctx,
			"verified outdated %#q HelmRepository CR, from %#q namespace, is gone",
			expectedHRToDelete.Name,
			expectedHRToDelete.Namespace,
		)
	}

	// Stage 3: remove Catalog CR and watch HemRepository CRs being deleted
	{
		config.Logger.Debugf(ctx, "deleting %#q Catalog CR", catalog.Name)

		err = config.K8sClients.CtrlClient().Delete(ctx, &currentCatalog)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "deleted %#q Catalog CR", catalog.Name)

		for _, entry := range currentCatalog.Status.HelmRepositoryList.Entries {
			config.Logger.Debugf(ctx, "verifying %#q HelmRepository CR, from %#q namespace, is gone", entry.Name, entry.Namespace)

			o := func() error {
				err = config.K8sClients.CtrlClient().Get(
					ctx,
					types.NamespacedName{Name: entry.Name, Namespace: entry.Namespace},
					&sourcev1.HelmRepository{},
				)
				if !apierrors.IsNotFound(err) {
					return microerror.Mask(err)
				}

				return nil
			}

			n := func(err error, t time.Duration) {
				config.Logger.Errorf(
					ctx,
					err,
					"failed to ensure %#q HelmRepository CR, is namespace %#q, is gone: retrying in %s",
					entry.Name,
					entry.Namespace,
					t,
				)
			}

			b := backoff.NewConstant(5*time.Minute, 15*time.Second)
			err := backoff.RetryNotify(o, b, n)
			if err != nil {
				t.Fatalf("expected %#v got %#v", nil, err)
			}

			config.Logger.Debugf(ctx, "verified %#q HelmRepository CR, from %#q namespace, is gone", entry.Name, entry.Namespace)
		}

		config.Logger.Debugf(ctx, "verifying %#q Catalog CR, from %#q namespace, is gone", catalog.Name, catalog.Namespace)

		o := func() error {
			err = config.K8sClients.CtrlClient().Get(
				ctx,
				types.NamespacedName{Name: catalog.Name, Namespace: catalog.Namespace},
				&v1alpha1.Catalog{},
			)
			if !apierrors.IsNotFound(err) {
				return microerror.Mask(err)
			}

			return nil
		}

		n := func(err error, t time.Duration) {
			config.Logger.Errorf(
				ctx,
				err,
				"failed to ensure %#q Catalog CR, is namespace %#q, is gone: retrying in %s",
				catalog.Name,
				catalog.Namespace,
				t,
			)
		}

		b := backoff.NewConstant(5*time.Minute, 15*time.Second)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		config.Logger.Debugf(ctx, "verified %#q Catalog CR, from %#q namespace, is gone", catalog.Name, catalog.Namespace)

	}

}

func createFromTemplate(template sourcev1.HelmRepository, source interface{}) (*sourcev1.HelmRepository, error) {
	hrType, hrURL, err := operatorkey.GetRepositoryConfiguration(source)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	hrName, err := operatorkey.GetHelmRepositoryName(template.Name, hrType, hrURL)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	template.Name = hrName
	template.Spec.URL = hrURL

	if hrType == "oci" {
		template.Spec.Type = hrType
	}

	return &template, nil
}

func getExpectedHRs(catalog *v1alpha1.Catalog) ([]*sourcev1.HelmRepository, error) {
	hrs := make([]*sourcev1.HelmRepository, 0)

	hrTemplate := sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      catalog.Name,
			Namespace: catalog.Namespace,
		},
		Spec: sourcev1.HelmRepositorySpec{
			Interval: metav1.Duration{Duration: 10 * time.Minute},
			Provider: "generic",
			Timeout:  &metav1.Duration{Duration: 1 * time.Minute},
		},
	}

	hr, err := createFromTemplate(hrTemplate, catalog.Spec.Storage)
	if err != nil {
		return []*sourcev1.HelmRepository{}, microerror.Mask(err)
	}

	hrs = append(
		hrs,
		hr,
	)

	for _, r := range catalog.Spec.Repositories {
		hr, err = createFromTemplate(hrTemplate, r)
		if err != nil {
			return []*sourcev1.HelmRepository{}, microerror.Mask(err)
		}

		hrs = append(
			hrs,
			hr,
		)
	}

	return hrs, nil
}
