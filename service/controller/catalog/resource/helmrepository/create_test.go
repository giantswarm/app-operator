package helmrepository

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck
)

func Test_createOrUpdateHelmRepositories(t *testing.T) {
	tests := []struct {
		existingObjects []*sourcev1.HelmRepository
		expectedObjects []sourcev1.HelmRepository
		expectedState   map[string]empty
		name            string
		object          v1alpha1.Catalog
	}{
		{
			expectedObjects: []sourcev1.HelmRepository{
				sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						Interval: metav1.Duration{Duration: 10 * time.Minute},
						Provider: "generic",
						Timeout:  &metav1.Duration{Duration: 1 * time.Minute},
						Type:     "",
						URL:      "https://giantswarm.github.io/app-catalog/",
					},
				},
				sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						Interval: metav1.Duration{Duration: 10 * time.Minute},
						Provider: "generic",
						Timeout:  &metav1.Duration{Duration: 1 * time.Minute},
						Type:     "oci",
						URL:      "oci://giantswarmpublic.azurecr.io/app-catalog/",
					},
				},
			},
			expectedState: map[string]empty{
				"giantswarm-helm-giantswarm.github.io-app-catalog":       empty{},
				"giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog": empty{},
			},
			name: "HelmRepository CRs creation",
			object: v1alpha1.Catalog{
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
		},
		{
			existingObjects: []*sourcev1.HelmRepository{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						Interval: metav1.Duration{Duration: 10 * time.Minute},
						Provider: "generic",
						Timeout:  &metav1.Duration{Duration: 1 * time.Minute},
						Type:     "",
						URL:      "https://giantswarm.github.io/app-catalog/",
					},
				},
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						Interval: metav1.Duration{Duration: 10 * time.Minute},
						Provider: "generic",
						Timeout:  &metav1.Duration{Duration: 1 * time.Minute},
						Type:     "oci",
						URL:      "oci://giantswarmpublic.azurecr.io/app-catalog/",
					},
				},
			},
			expectedObjects: []sourcev1.HelmRepository{
				sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						Interval: metav1.Duration{Duration: 10 * time.Minute},
						Provider: "generic",
						Timeout:  &metav1.Duration{Duration: 1 * time.Minute},
						Type:     "",
						URL:      "https://giantswarm.github.io/app-catalog/",
					},
				},
				sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						Interval: metav1.Duration{Duration: 10 * time.Minute},
						Provider: "generic",
						Timeout:  &metav1.Duration{Duration: 1 * time.Minute},
						Type:     "oci",
						URL:      "oci://giantswarmpublic.azurecr.io/app-catalog/",
					},
				},
			},
			expectedState: map[string]empty{
				"giantswarm-helm-giantswarm.github.io-app-catalog":       empty{},
				"giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog": empty{},
			},
			name: "HelmRepository CRs update",
			object: v1alpha1.Catalog{
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
		},
		{
			existingObjects: []*sourcev1.HelmRepository{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"test_key": "test_value",
						},
						Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						Interval: metav1.Duration{Duration: 100 * time.Minute},
						Provider: "generic",
						Timeout:  &metav1.Duration{Duration: 100 * time.Minute},
						Type:     "",
						URL:      "https://giantswarm.github.io/app-catalog/",
					},
				},
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						Interval: metav1.Duration{Duration: 100 * time.Minute},
						Provider: "generic",
						Timeout:  &metav1.Duration{Duration: 100 * time.Minute},
						Type:     "oci",
						URL:      "oci://giantswarmpublic.azurecr.io/app-catalog/",
					},
				},
			},
			expectedObjects: []sourcev1.HelmRepository{
				sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-helm-giantswarm.github.io-app-catalog",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						Interval: metav1.Duration{Duration: 10 * time.Minute},
						Provider: "generic",
						Timeout:  &metav1.Duration{Duration: 1 * time.Minute},
						Type:     "",
						URL:      "https://giantswarm.github.io/app-catalog/",
					},
				},
				sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						Interval: metav1.Duration{Duration: 10 * time.Minute},
						Provider: "generic",
						Timeout:  &metav1.Duration{Duration: 1 * time.Minute},
						Type:     "oci",
						URL:      "oci://giantswarmpublic.azurecr.io/app-catalog/",
					},
				},
			},
			expectedState: map[string]empty{
				"giantswarm-helm-giantswarm.github.io-app-catalog":       empty{},
				"giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog": empty{},
			},
			name: "HelmRepository CRs restore of desired values",
			object: v1alpha1.Catalog{
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

			state, err := r.createOrUpdateHelmRepositories(context.Background(), tc.object)
			if err != nil {
				t.Fatalf("error == %#v, want <nil>", err)
			}

			if !reflect.DeepEqual(state, tc.expectedState) {
				t.Fatalf("want matching states \n %s", cmp.Diff(state, tc.expectedState))
			}

			for _, desiredHR := range tc.expectedObjects {
				currentHR := &sourcev1.HelmRepository{}
				err = r.ctrlClient.Get(
					context.Background(),
					types.NamespacedName{Name: desiredHR.Name, Namespace: desiredHR.Namespace},
					currentHR,
				)

				if err != nil {
					t.Fatalf("error == %#v, want <nil>", err)
				}

				if !reflect.DeepEqual(currentHR.Labels, desiredHR.Labels) {
					t.Fatalf("want matching labels \n %s", cmp.Diff(currentHR.Labels, desiredHR.Labels))
				}

				if !reflect.DeepEqual(currentHR.Annotations, desiredHR.Annotations) {
					t.Fatalf("want matching annotations \n %s", cmp.Diff(currentHR.Annotations, desiredHR.Annotations))
				}

				if !reflect.DeepEqual(currentHR.Spec, desiredHR.Spec) {
					t.Fatalf("want matching spec \n %s", cmp.Diff(currentHR.Spec, desiredHR.Spec))
				}
			}
		})
	}
}

func Test_deleteGoneHelmRepositories(t *testing.T) {
	tests := []struct {
		currentState     *v1alpha1.HelmRepositoryList
		desiredState     map[string]empty
		existingObjects  []*sourcev1.HelmRepository
		expectedToDelete []types.NamespacedName
		name             string
	}{
		{
			currentState: &v1alpha1.HelmRepositoryList{
				Entries: []v1alpha1.HelmRepositoryRef{},
			},
			desiredState: map[string]empty{
				"giantswarm-helm-giantswarm.github.io-app-catalog":       empty{},
				"giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog": empty{},
			},
			name: "HelmRepository CRs creation",
		},
		{
			currentState: &v1alpha1.HelmRepositoryList{
				Entries: []v1alpha1.HelmRepositoryRef{
					v1alpha1.HelmRepositoryRef{
						Name:      "giantswarm-helm-giantswarm.github.io-app-catalog-test",
						Namespace: "default",
					},
				},
			},
			desiredState: map[string]empty{
				"giantswarm-helm-giantswarm.github.io-app-catalog":       empty{},
				"giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog": empty{},
			},
			existingObjects: []*sourcev1.HelmRepository{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-helm-giantswarm.github.io-app-catalog-test",
						Namespace: "default",
					},
				},
			},
			expectedToDelete: []types.NamespacedName{
				types.NamespacedName{
					Name:      "giantswarm-helm-giantswarm.github.io-app-catalog-test",
					Namespace: "default",
				},
			},
			name: "HelmRepository CRs update",
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

			err = r.deleteGoneHelmRepositories(context.Background(), tc.desiredState, tc.currentState)
			if err != nil {
				t.Fatalf("error == %#v, want <nil>", err)
			}

			for _, hr := range tc.expectedToDelete {
				err = r.ctrlClient.Get(
					context.Background(),
					hr,
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

func Test_updateCatalogStatus(t *testing.T) {
	tests := []struct {
		catalog        *v1alpha1.Catalog
		desiredState   map[string]empty
		expectedStatus map[string]string
		name           string
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
			desiredState: map[string]empty{
				"giantswarm-helm-giantswarm.github.io-app-catalog":       empty{},
				"giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog": empty{},
			},
			expectedStatus: map[string]string{
				"giantswarm-helm-giantswarm.github.io-app-catalog":       "default",
				"giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog": "default",
			},
			name: "HelmRepository CRs creation",
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
							v1alpha1.HelmRepositoryRef{
								Name:      "giantswarm-helm-giantswarm.github.io-app-catalog-test",
								Namespace: "default",
							},
						},
					},
				},
			},
			desiredState: map[string]empty{
				"giantswarm-helm-giantswarm.github.io-app-catalog":       empty{},
				"giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog": empty{},
			},
			expectedStatus: map[string]string{
				"giantswarm-helm-giantswarm.github.io-app-catalog":       "default",
				"giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog": "default",
			},
			name: "HelmRepository CRs update",
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)
			_ = sourcev1.AddToScheme(scheme)

			objs := []runtime.Object{tc.catalog}

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

			err = r.updateCatalogStatus(context.Background(), *tc.catalog, tc.desiredState)
			if err != nil {
				t.Fatalf("error == %#v, want <nil>", err)
			}

			ct := &v1alpha1.Catalog{}
			err = r.ctrlClient.Get(
				context.Background(),
				types.NamespacedName{Name: "giantswarm", Namespace: "default"},
				ct,
			)
			if err != nil {
				t.Fatalf("error == %#v, want <nil>", err)
			}

			if ct.Status.HelmRepositoryList == nil {
				t.Fatal("got <nil>, want not <nil>")
			}

			if len(ct.Status.HelmRepositoryList.Entries) != len(tc.expectedStatus) {
				t.Fatalf(
					"got '%d' vs '%d', want equal",
					len(ct.Status.HelmRepositoryList.Entries),
					len(tc.expectedStatus),
				)
			}

			for _, e := range ct.Status.HelmRepositoryList.Entries {
				if e.Namespace != tc.expectedStatus[e.Name] {
					t.Fatalf(
						"got '%s' vs '%s', want matching",
						e.Namespace,
						tc.expectedStatus[e.Name],
					)
				}
			}
		})
	}
}
