package values

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_MergeConfigMapData(t *testing.T) {
	tests := []struct {
		name         string
		appData      map[string]string
		catalogData  map[string]string
		expectedData map[string]string
		errorMatcher func(error) bool
	}{
		{
			name:         "case 0: empty data",
			appData:      map[string]string{},
			catalogData:  map[string]string{},
			expectedData: map[string]string{},
		},
		{
			name: "case 1: only app data",
			appData: map[string]string{
				"values": "test: yaml",
			},
			catalogData: map[string]string{},
			expectedData: map[string]string{
				"values": "test: yaml",
			},
		},
		{
			name:    "case 2: only catalog data",
			appData: map[string]string{},
			catalogData: map[string]string{
				"values": "test: yaml",
			},
			expectedData: map[string]string{
				"values": "test: yaml",
			},
		},
		{
			name: "case 3: clashing app and catalog data",
			appData: map[string]string{
				"values": "test: yaml",
			},
			catalogData: map[string]string{
				"values": "test: yaml",
			},
			errorMatcher: IsFailedExecution,
		},
		{
			name: "case 4: merged simple app and catalog data",
			appData: map[string]string{
				"values": "app: data",
			},
			catalogData: map[string]string{
				"values": "catalog: data",
			},
			expectedData: map[string]string{
				"values": "app: data\ncatalog: data\n",
			},
		},
		{
			name: "case 5: different app and catalog keys",
			appData: map[string]string{
				"app": "app: data",
			},
			catalogData: map[string]string{
				"catalog": "catalog: data",
			},
			expectedData: map[string]string{
				"app":     "app: data",
				"catalog": "catalog: data",
			},
		},
		{
			name: "case 6: multiple app keys and single catalog key",
			appData: map[string]string{
				"extra":  "app: data",
				"values": "app: data",
			},
			catalogData: map[string]string{
				"values": "catalog: data",
			},
			expectedData: map[string]string{
				"extra":  "app: data",
				"values": "app: data\ncatalog: data\n",
			},
		},
		{
			name: "case 7: single app key and multiple catalog keys",
			appData: map[string]string{
				"values": "app: data",
			},
			catalogData: map[string]string{
				"extra":  "catalog: data",
				"values": "catalog: data",
			},
			expectedData: map[string]string{
				"extra":  "catalog: data",
				"values": "app: data\ncatalog: data\n",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := MergeConfigMapData(tc.appData, tc.catalogData)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if !reflect.DeepEqual(result, tc.expectedData) {
				t.Fatalf("want matching \n %s", cmp.Diff(result, tc.expectedData))
			}
		})
	}
}
