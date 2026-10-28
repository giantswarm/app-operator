package appcatalogentry

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v8/pkg/key"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/app-operator/v7/pkg/project"
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

func Test_EnsureCreated_disableEntries(t *testing.T) {
	newEntry := func(name, catalog, managedBy string) *v1alpha1.AppCatalogEntry {
		return &v1alpha1.AppCatalogEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: metav1.NamespaceDefault,
				Labels: map[string]string{
					label.CatalogName: catalog,
					label.ManagedBy:   managedBy,
				},
			},
		}
	}

	managedBy := key.AppCatalogEntryManagedBy(project.Name())

	catalog := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "giantswarm",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1alpha1.CatalogSpec{
			// Unreachable: the test fails if the index.yaml is fetched.
			Storage: v1alpha1.CatalogSpecStorage{URL: "http://127.0.0.1:1/", Type: "helm"},
		},
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.AppCatalogEntry{}, &v1alpha1.AppCatalogEntryList{})

	ctrlClient := fake.NewClientBuilder().WithScheme(s).WithObjects(
		newEntry("giantswarm-foo-1.0.0", "giantswarm", managedBy),
		newEntry("giantswarm-foo-1.1.0", "giantswarm", managedBy),
		newEntry("other-foo-1.0.0", "other", managedBy),
		newEntry("giantswarm-manual-1.0.0", "giantswarm", "someone-else"),
	).Build()

	r, err := New(Config{
		K8sClient: k8sclienttest.NewClients(k8sclienttest.ClientsConfig{CtrlClient: ctrlClient}),
		Logger:    microloggertest.New(),

		DisableEntries: true,
		Provider:       "aws",
		UniqueApp:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	err = r.EnsureCreated(context.Background(), catalog)
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	var entries v1alpha1.AppCatalogEntryList
	err = ctrlClient.List(context.Background(), &entries)
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	var remaining []string
	for _, e := range entries.Items {
		remaining = append(remaining, e.Name)
	}
	sort.Strings(remaining)

	// Only the entries this catalog's resource manages are deleted.
	expected := []string{"giantswarm-manual-1.0.0", "other-foo-1.0.0"}
	if !reflect.DeepEqual(remaining, expected) {
		t.Fatalf("remaining entries = %v, want %v", remaining, expected)
	}
}
