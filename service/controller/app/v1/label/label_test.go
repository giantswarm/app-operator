package label

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_FilterLabels(t *testing.T) {
	tests := []struct {
		name           string
		appLabels      map[string]string
		excludeLabels  map[string]string
		requiredLabels map[string]string
		expectedLabels map[string]string
	}{
		{
			name: "case 0: basic match",
			appLabels: map[string]string{
				"app-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":           "release-operator",
			},
			expectedLabels: map[string]string{
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":             "app-operator",
			},
		},
		{
			name: "case 1: extra labels still present",
			appLabels: map[string]string{
				"app": "prometheus",
				"app-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/cluster":              "5xchu",
				"giantswarm.io/managed-by":           "cluster-operator",
				"giantswarm.io/organization":         "giantswarm",
			},
			expectedLabels: map[string]string{
				"app": "prometheus",
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/cluster":                "5xchu",
				"giantswarm.io/managed-by":             "app-operator",
				"giantswarm.io/organization":           "giantswarm",
			},
		},
		{
			name: "case 2: empty inputs",
			expectedLabels: map[string]string{
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":             "app-operator",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			result := FilterLabels(tc.appLabels, tc.excludeLabels, tc.requiredLabels)

			if !reflect.DeepEqual(result, tc.expectedLabels) {
				t.Fatalf("want matching labels \n %s", cmp.Diff(result, tc.expectedLabels))
			}
		})
	}
}
