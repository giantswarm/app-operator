package key

import (
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
					Storage: v1alpha1.AppCatalogSpecStorage{
						Type: "helm",
						URL:  "http://giantswarm.io/sample-catalog.tgz",
					},
				},
			},
			expectedObject: v1alpha1.AppCatalog{
				Spec: v1alpha1.AppCatalogSpec{
					Title:       "giant-swarm-title",
					Description: "giant-swarm app catalog sample",
					Storage: v1alpha1.AppCatalogSpecStorage{
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

func Test_VersionLabel(t *testing.T) {
	testCases := []struct {
		name            string
		obj             v1alpha1.AppCatalog
		expectedVersion string
		errorMatcher    func(error) bool
	}{
		{
			name: "case 0: basic match",
			obj: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
			},
			expectedVersion: "1.0.0",
		},
		{
			name: "case 1: incorrect label",
			obj: v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"chart-operator.giantswarm.io/version": "1.0.0",
					},
				},
			},
			expectedVersion: "",
		},
		{
			name:            "case 2: no labels",
			obj:             v1alpha1.AppCatalog{},
			expectedVersion: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := VersionLabel(tc.obj)

			if !reflect.DeepEqual(result, tc.expectedVersion) {
				t.Fatalf("Version label == %#v, want %#v", result, tc.expectedVersion)
			}
		})
	}
}
