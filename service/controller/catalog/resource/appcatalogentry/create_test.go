package appcatalogentry

import (
	"context"
	"testing"
	"time"

	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getLatestEntry(t *testing.T) {
	day := func(d int) metav1.Time {
		return metav1.NewTime(time.Date(2026, 9, d, 0, 0, 0, 0, time.UTC))
	}

	tests := []struct {
		name     string
		entries  []entry
		expected string
	}{
		{
			name: "highest stable version",
			entries: []entry{
				{Version: "1.1.0", Created: day(3)},
				{Version: "1.2.0", Created: day(2)},
				{Version: "1.0.0", Created: day(1)},
			},
			expected: "1.2.0",
		},
		{
			name: "a newer release candidate does not replace the stable version",
			entries: []entry{
				{Version: "1.3.0-rc.1", Created: day(3)},
				{Version: "1.2.0", Created: day(2)},
				{Version: "1.1.0", Created: day(1)},
			},
			expected: "1.2.0",
		},
		{
			name: "a candidate of the same core version does not replace the stable version",
			entries: []entry{
				{Version: "1.2.0-rc.2", Created: day(3)},
				{Version: "1.2.0", Created: day(2)},
			},
			expected: "1.2.0",
		},
		{
			name: "the stable version supersedes its candidates",
			entries: []entry{
				{Version: "1.3.0", Created: day(3)},
				{Version: "1.3.0-rc.1", Created: day(2)},
				{Version: "1.2.0", Created: day(1)},
			},
			expected: "1.3.0",
		},
		{
			name: "without a stable version the pre-releases are compared, the newest wins a tie",
			entries: []entry{
				{Version: "1.3.0-8a3f2c1", Created: day(3)},
				{Version: "1.3.0-2b7e9d4", Created: day(2)},
				{Version: "1.2.0-c0ffee0", Created: day(1)},
			},
			expected: "1.3.0-8a3f2c1",
		},
		{
			name: "invalid versions are skipped",
			entries: []entry{
				{Version: "not-a-version", Created: day(3)},
				{Version: "1.2.0", Created: day(2)},
			},
			expected: "1.2.0",
		},
		{
			name: "without any valid version the newest entry",
			entries: []entry{
				{Version: "not-a-version", Created: day(3)},
				{Version: "neither", Created: day(2)},
			},
			expected: "not-a-version",
		},
	}

	r := &Resource{logger: microloggertest.New()}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			latest, err := r.getLatestEntry(context.Background(), tc.entries)
			if err != nil {
				t.Fatalf("unexpected error: %#v", err)
			}
			if latest.Version != tc.expected {
				t.Fatalf("latest version = %q, want %q", latest.Version, tc.expected)
			}
		})
	}
}
