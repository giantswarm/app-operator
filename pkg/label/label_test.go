package label

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_processLabels(t *testing.T) {
	tests := []struct {
		name           string
		requiredLabels map[string]string
		excludeLabels  map[string]string
		inputLabels    map[string]string
		expectedLabels map[string]string
	}{
		{
			name: "case 0: basic match",
			requiredLabels: map[string]string{
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":             "app-operator",
			},
			inputLabels: map[string]string{
				"giantswarm.io/managed-by": "release-operator",
			},
			expectedLabels: map[string]string{
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":             "app-operator",
			},
		},
		{
			name: "case 1: extra labels still present",
			requiredLabels: map[string]string{
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/managed-by":             "app-operator",
			},
			inputLabels: map[string]string{
				"app":                        "prometheus",
				"giantswarm.io/cluster":      "5xchu",
				"giantswarm.io/managed-by":   "cluster-operator",
				"giantswarm.io/organization": "giantswarm",
			},
			expectedLabels: map[string]string{
				"app":                                  "prometheus",
				"chart-operator.giantswarm.io/version": "1.0.0",
				"giantswarm.io/cluster":                "5xchu",
				"giantswarm.io/managed-by":             "app-operator",
				"giantswarm.io/organization":           "giantswarm",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			result := ProcessLabels(tc.inputLabels, tc.requiredLabels, tc.excludeLabels)

			if !reflect.DeepEqual(result, tc.expectedLabels) {
				t.Fatalf("want matching \n %s", cmp.Diff(result, tc.expectedLabels))
			}
		})
	}
}
