package values

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	testComplexYaml = `
Installation:
  V1:
    Provider:
      AWS:
        AvailabilityZone: 'eu-central-1a'
        Region: 'eu-central-1'
        Route53:
          HostedZones:
            API: 'Z1...'
            Etcd: 'Z1...'
            Ingress: 'Z1...'
        VPCPeerID: 'vpc-123'`

	testExtraYaml = `
Test:
  V1:
    Provider:
      Aws:
        Region: 'eu-west-1'`

	// TODO Fix bug with stipped quotes from YAML.
	testMergedYaml = `Installation:
  V1:
    Provider:
      AWS:
        AvailabilityZone: eu-central-1a
        Region: eu-central-1
        Route53:
          HostedZones:
            API: Z1...
            Etcd: Z1...
            Ingress: Z1...
        VPCPeerID: vpc-123
Test:
  V1:
    Provider:
      Aws:
        Region: eu-west-1`
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
			name:         "case 0: empty data doesn't error",
			expectedData: map[string]string{},
		},
		{
			name:         "case 1: empty data",
			appData:      map[string]string{},
			catalogData:  map[string]string{},
			expectedData: map[string]string{},
		},
		{
			name: "case 2: only app data",
			appData: map[string]string{
				"values": "test: yaml",
			},
			catalogData: map[string]string{},
			expectedData: map[string]string{
				"values": "test: yaml",
			},
		},
		{
			name:    "case 3: only catalog data",
			appData: map[string]string{},
			catalogData: map[string]string{
				"values": "test: yaml",
			},
			expectedData: map[string]string{
				"values": "test: yaml",
			},
		},
		{
			name: "case 4: clashing app and catalog data",
			appData: map[string]string{
				"values": "test: yaml",
			},
			catalogData: map[string]string{
				"values": "test: yaml",
			},
			errorMatcher: IsFailedExecution,
		},
		{
			name: "case 5: merged simple app and catalog data",
			appData: map[string]string{
				"values": "app: data",
			},
			catalogData: map[string]string{
				"values": "catalog: data",
			},
			expectedData: map[string]string{
				"values": "app: data\ncatalog: data",
			},
		},
		{
			name: "case 6: different app and catalog keys",
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
			name: "case 7: multiple app keys and single catalog key",
			appData: map[string]string{
				"extra":  "app: data",
				"values": "app: data",
			},
			catalogData: map[string]string{
				"values": "catalog: data",
			},
			expectedData: map[string]string{
				"extra":  "app: data",
				"values": "app: data\ncatalog: data",
			},
		},
		{
			name: "case 8: single app key and multiple catalog keys",
			appData: map[string]string{
				"values": "app: data",
			},
			catalogData: map[string]string{
				"extra":  "catalog: data",
				"values": "catalog: data",
			},
			expectedData: map[string]string{
				"extra":  "catalog: data",
				"values": "app: data\ncatalog: data",
			},
		},
		{
			name: "case 9: complex app yaml and no catalog keys",
			appData: map[string]string{
				"values": testComplexYaml,
			},
			expectedData: map[string]string{
				"values": testComplexYaml,
			},
		},
		{
			name: "case 10: complex catalog yaml and no app keys",
			catalogData: map[string]string{
				"values": testComplexYaml,
			},
			expectedData: map[string]string{
				"values": testComplexYaml,
			},
		},
		{
			name: "case 11: complex clashing yaml",
			appData: map[string]string{
				"values": testComplexYaml,
			},
			catalogData: map[string]string{
				"values": testComplexYaml,
			},
			errorMatcher: IsFailedExecution,
		},
		{
			// TODO Fix bug with stipped quotes from YAML.
			name: "case 12: merge yaml",
			appData: map[string]string{
				"values": testComplexYaml,
			},
			catalogData: map[string]string{
				"values": testExtraYaml,
			},
			expectedData: map[string]string{
				"values": testMergedYaml,
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
