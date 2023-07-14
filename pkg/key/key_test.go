package key

import (
	"fmt"
	"testing"
)

func Test_RepositoryName(t *testing.T) {
	tests := []struct {
		catalog       string
		expectedError func(error) bool
		expectedName  string
		name          string
		repoType      string
		repoURL       string
	}{
		{
			catalog:       "giantswarm",
			expectedError: func(e error) bool { return e == nil },
			expectedName:  "giantswarm-helm-giantswarm.github.io-app-catalog",
			name:          "flawless primary Helm repository",
			repoType:      "helm",
			repoURL:       "https://giantswarm.github.io/app-catalog",
		},
		{
			catalog:       "giantswarm",
			expectedError: func(e error) bool { return e == nil },
			expectedName:  "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
			name:          "flawless primary OCI repository",
			repoType:      "oci",
			repoURL:       "oci://giantswarmpublic.azurecr.io/app-catalog/",
		},
		{
			catalog:       "giantswarm",
			expectedError: func(e error) bool { return e == nil },
			expectedName:  "giantswarm-helm-giantswarm.github.io-app-catalog",
			name:          "flawless fallback Helm repository",
			repoType:      "helm",
			repoURL:       "https://giantswarm.github.io/app-catalog",
		},
		{
			catalog:       "giantswarm",
			expectedError: func(e error) bool { return e == nil },
			expectedName:  "giantswarm-oci-giantswarmpublic.azurecr.io-app-catalog",
			name:          "flawless fallback OCI repository",
			repoType:      "oci",
			repoURL:       "oci://giantswarmpublic.azurecr.io/app-catalog/",
		},
	}

	for c, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", c, tc.name), func(t *testing.T) {
			name, err := GetHelmRepositoryName(tc.catalog, tc.repoType, tc.repoURL)

			if name != tc.expectedName {
				t.Fatalf("got %s, want %s", tc.expectedName, name)
			}

			if !tc.expectedError(err) {
				t.Fatalf("got wrong error %#v", err)
			}
		})
	}
}
