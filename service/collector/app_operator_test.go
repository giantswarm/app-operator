package collector

import (
	"strconv"
	"testing"
)

func Test_helm2AppOperatorReady(t *testing.T) {
	tests := []struct {
		name             string
		operatorVersions map[string]int32
		expectedResult   int32
		hasError         bool
	}{
		{
			name: "case 0: correct operator versions",
			operatorVersions: map[string]int32{
				"0.0.0": 1,
				"1.0.9": 1,
				"2.0.0": 1,
			},
			expectedResult: 1,
		},
		{
			name: "case 1: helm 2 operator not ready",
			operatorVersions: map[string]int32{
				"0.0.0": 1,
				"1.0.9": 0,
				"2.0.0": 1,
			},
			expectedResult: 0,
		},
		{
			name: "case 2: helm 2 operator missing",
			operatorVersions: map[string]int32{
				"0.0.0": 1,
				"2.0.0": 1,
			},
			expectedResult: 0,
		},
		{
			name: "case 3: multiple helm 2 operators",
			operatorVersions: map[string]int32{
				"0.0.0": 0,
				"1.0.9": 1,
				"1.1.0": 1,
				"2.0.0": 0,
			},
			expectedResult: 2,
		},
		{
			name:             "case 4: no versions",
			operatorVersions: map[string]int32{},
			expectedResult:   0,
		},
	}

	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			result, err := helm2AppOperatorReady(tc.operatorVersions)

			switch {
			case err == nil && tc.hasError == false:
				// correct; carry on
			case err != nil && tc.hasError == false:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.hasError:
				t.Fatalf("error == nil, want non-nil")
			}

			if result != tc.expectedResult {
				t.Fatalf("expected %d, got %d", result, tc.expectedResult)
			}
		})
	}
}
