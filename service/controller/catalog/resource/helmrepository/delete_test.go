package helmrepository

import (
	"context"
	"fmt"
	"testing"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck
)

func Test_EnsureDeleted(t *testing.T) {
	tests := []struct {
		catalog         *v1alpha1.Catalog
		existingObjects []*sourcev1.HelmRepository
		expectedGone    []types.NamespacedName
		name            string
	}{
		{
			catalog: &v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.io/app-catalog/",
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "https://giantswarm.github.io/app-catalog/",
						},
						{
							Type: "oci",
							URL:  "oci://giantswarmpublic.azurecr.io/app-catalog/",
						},
					},
					LogoURL: "https://s.giantswarm.io/giantswarm.png",
				},
			},
			expectedGone: []types.NamespacedName{
				types.NamespacedName{
					Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
					Namespace: "default",
				},
				types.NamespacedName{
					Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
					Namespace: "default",
				},
			},
			name: "HelmRepository CRs deletion with no status",
		},
		{
			catalog: &v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.io/app-catalog/",
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "https://giantswarm.github.io/app-catalog/",
						},
						{
							Type: "oci",
							URL:  "oci://giantswarmpublic.azurecr.io/app-catalog/",
						},
					},
					LogoURL: "https://s.giantswarm.io/giantswarm.png",
				},
				Status: v1alpha1.CatalogStatus{
					HelmRepositoryList: &v1alpha1.HelmRepositoryList{
						Entries: []v1alpha1.HelmRepositoryRef{
							v1alpha1.HelmRepositoryRef{
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog-test",
								Namespace: "default",
							},
						},
					},
				},
			},
			existingObjects: []*sourcev1.HelmRepository{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-helm-giantswarm.github.io-app-catalog-test",
						Namespace: "default",
					},
				},
			},
			expectedGone: []types.NamespacedName{
				types.NamespacedName{
					Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
					Namespace: "default",
				},
				types.NamespacedName{
					Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
					Namespace: "default",
				},
				types.NamespacedName{
					Name:      "giantswarm-helm-giantswarm.github.io-app-catalog-test",
					Namespace: "default",
				},
			},
			name: "HelmRepository CRs deletion with old status",
		},
		{
			catalog: &v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
				},
				Spec: v1alpha1.CatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.CatalogSpecStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.io/app-catalog/",
					},
					Repositories: []v1alpha1.CatalogSpecRepository{
						{
							Type: "helm",
							URL:  "https://giantswarm.github.io/app-catalog/",
						},
						{
							Type: "oci",
							URL:  "oci://giantswarmpublic.azurecr.io/app-catalog/",
						},
					},
					LogoURL: "https://s.giantswarm.io/giantswarm.png",
				},
				Status: v1alpha1.CatalogStatus{
					HelmRepositoryList: &v1alpha1.HelmRepositoryList{
						Entries: []v1alpha1.HelmRepositoryRef{
							v1alpha1.HelmRepositoryRef{
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
								Namespace: "default",
							},
							v1alpha1.HelmRepositoryRef{
								Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
								Namespace: "default",
							},
						},
					},
				},
			},
			existingObjects: []*sourcev1.HelmRepository{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
						Namespace: "default",
					},
				},
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
						Namespace: "default",
					},
				},
			},
			expectedGone: []types.NamespacedName{
				types.NamespacedName{
					Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
					Namespace: "default",
				},
				types.NamespacedName{
					Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
					Namespace: "default",
				},
			},
			name: "HelmRepository CRs deletion with up to date status",
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = sourcev1.AddToScheme(scheme)

			objs := make([]runtime.Object, 0)
			for _, o := range tc.existingObjects {
				objs = append(objs, o)
			}

			c := Config{
				CtrlClient: fake.NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(objs...).
					Build(),
				Logger: microloggertest.New(),
			}

			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want <nil>", err)
			}

			err = r.EnsureDeleted(context.Background(), tc.catalog)
			if err != nil {
				t.Fatalf("error == %#v, want <nil>", err)
			}

			for _, object := range tc.expectedGone {
				err = r.ctrlClient.Get(
					context.Background(),
					object,
					&sourcev1.HelmRepository{},
				)

				if err == nil {
					t.Fatal("got <nil>, want 'NotFoundError'")
				}
				if !apierrors.IsNotFound(err) {
					t.Fatalf("error == %#v, want 'NotFoundError'", err)
				}
			}
		})
	}
}
