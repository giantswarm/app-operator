package appcatalogentry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getDesiredAppCatalogEntryCR_metadataAnnotation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("chartApiVersion: v2\nupstreamChartVersion: 1.2.3\n"))
	}))
	defer srv.Close()

	tests := []struct {
		name               string
		annotations        map[string]string
		expectedAPIVersion string
		expectedUpstream   string
	}{
		{
			name:               "metadata URL in the current catalog annotation",
			annotations:        map[string]string{"io.giantswarm.application.metadata": srv.URL},
			expectedAPIVersion: "v2",
			expectedUpstream:   "1.2.3",
		},
		{
			name:               "metadata URL in the legacy catalog annotation",
			annotations:        map[string]string{"application.giantswarm.io/metadata": srv.URL},
			expectedAPIVersion: "v2",
			expectedUpstream:   "1.2.3",
		},
		{
			name:               "no metadata URL falls back to chart API version v1",
			annotations:        map[string]string{},
			expectedAPIVersion: "v1",
			expectedUpstream:   "",
		},
	}

	r := &Resource{logger: microloggertest.New()}
	cr := &v1alpha1.Catalog{ObjectMeta: metav1.ObjectMeta{Name: "giantswarm", Namespace: "default"}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := entry{
				Annotations: tc.annotations,
				Created:     metav1.Now(),
				Name:        "hello-world",
				Version:     "3.2.4",
			}

			entryCR, err := r.getDesiredAppCatalogEntryCR(context.Background(), cr, e, true)
			if err != nil {
				t.Fatalf("unexpected error: %#v", err)
			}
			if entryCR.Spec.Chart.APIVersion != tc.expectedAPIVersion {
				t.Fatalf("chart API version = %q, want %q", entryCR.Spec.Chart.APIVersion, tc.expectedAPIVersion)
			}
			if entryCR.Spec.Chart.UpstreamChartVersion != tc.expectedUpstream {
				t.Fatalf("upstream chart version = %q, want %q", entryCR.Spec.Chart.UpstreamChartVersion, tc.expectedUpstream)
			}
		})
	}
}
