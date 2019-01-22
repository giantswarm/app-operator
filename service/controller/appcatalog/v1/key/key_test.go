package key

import (
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_AppCatalogTitle(t *testing.T) {
	expectedName := "giant-swarm-title"

	obj := v1alpha1.AppCatalog{
		Spec: v1alpha1.AppCatalogSpec{
			Title:       "giant-swarm-title",
			Description: "giant-swarm app catalog sample",
			CatalogStorage: v1alpha1.AppCatalogSpecCatalogStorage{
				Type: "helm",
				URL:  "http://giantswarm.io/sample-catalog.tgz",
			},
		},
	}

	if AppCatalogTitle(obj) != expectedName {
		t.Fatalf("app catalog name %s, want %s", AppCatalogTitle(obj), expectedName)
	}
}

func Test_CatalogStorageURL(t *testing.T) {
	expectedName := "http://giantswarm.io/sample-catalog/"

	obj := v1alpha1.AppCatalog{
		Spec: v1alpha1.AppCatalogSpec{
			Title:       "giant-swarm-title",
			Description: "giant-swarm app catalog sample",
			CatalogStorage: v1alpha1.AppCatalogSpecCatalogStorage{
				Type: "helm",
				URL:  "http://giantswarm.io/sample-catalog/",
			},
		},
	}

	if CatalogStorageURL(obj) != expectedName {
		t.Fatalf("app catalog storage url %s, want %s", CatalogStorageURL(obj), expectedName)
	}
}

func Test_ConfigMapName(t *testing.T) {
	expectedName := "app-catalog-values"

	obj := v1alpha1.AppCatalog{
		Spec: v1alpha1.AppCatalogSpec{
			Title: "app-catalog",
			Config: v1alpha1.AppCatalogSpecConfig{
				ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
					Name:      "app-catalog-values",
					Namespace: "default",
				},
			},
		},
	}

	if ConfigMapName(obj) != expectedName {
		t.Fatalf("configmap name %#q, want %#q", ConfigMapName(obj), expectedName)
	}
}

func Test_ConfigMapNamespace(t *testing.T) {
	expectedNamespace := "default"

	obj := v1alpha1.AppCatalog{
		Spec: v1alpha1.AppCatalogSpec{
			Title: "app-catalog",
			Config: v1alpha1.AppCatalogSpecConfig{
				ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
					Name:      "app-catalog-values",
					Namespace: "default",
				},
			},
		},
	}

	if ConfigMapNamespace(obj) != expectedNamespace {
		t.Fatalf("configMap namespace %#q, want %#q", ConfigMapNamespace(obj), expectedNamespace)
	}
}

func Test_ToCustomResource(t *testing.T) {
	testCases := []struct {
		name           string
		input          interface{}
		expectedObject v1alpha1.AppCatalog
		errorMatcher   func(error) bool
	}{
		{
			name: "case 0: basic match",
			input: &v1alpha1.AppCatalog{
				Spec: v1alpha1.AppCatalogSpec{
					Title:       "giant-swarm-title",
					Description: "giant-swarm app catalog sample",
					CatalogStorage: v1alpha1.AppCatalogSpecCatalogStorage{
						Type: "helm",
						URL:  "http://giantswarm.io/sample-catalog.tgz",
					},
				},
			},
			expectedObject: v1alpha1.AppCatalog{
				Spec: v1alpha1.AppCatalogSpec{
					Title:       "giant-swarm-title",
					Description: "giant-swarm app catalog sample",
					CatalogStorage: v1alpha1.AppCatalogSpecCatalogStorage{
						Type: "helm",
						URL:  "http://giantswarm.io/sample-catalog.tgz",
					},
				},
			},
		},
		{
			name:         "case 1: wrong type",
			input:        &v1alpha1.App{},
			errorMatcher: IsWrongTypeError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ToCustomResource(tc.input)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if !reflect.DeepEqual(result, tc.expectedObject) {
				t.Fatalf("Custom Object == %#v, want %#v", result, tc.expectedObject)
			}
		})
	}
}

func TestVersionBundleVersion(t *testing.T) {
	testCases := []struct {
		name           string
		input          v1alpha1.AppCatalog
		expectedObject string
		errorMatcher   func(error) bool
	}{
		{
			name: "case 0: basic match",
			input: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"giantswarm.io/version-bundle": "0.1.0",
					},
				},
				Spec: v1alpha1.AppCatalogSpec{},
			},
			expectedObject: "0.1.0",
		},
		{
			name: "case 1: can't find key",
			input: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"giantswarm.io/version": "",
					},
				},
				Spec: v1alpha1.AppCatalogSpec{},
			},
			expectedObject: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := VersionBundleVersion(tc.input)

			if !reflect.DeepEqual(result, tc.expectedObject) {
				t.Fatalf("version == %#v, want %#v", result, tc.expectedObject)
			}
		})
	}
}
