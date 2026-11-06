package appcatalogentry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v8/pkg/key"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pkglabel "github.com/giantswarm/app-operator/v7/pkg/label"
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

func Test_New_rejectsNegativeMaxEntriesPerApp(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		_, err := New(Config{
			K8sClient: k8sclienttest.NewClients(k8sclienttest.ClientsConfig{}),
			Logger:    microloggertest.New(),

			DisableEntries:   disabled,
			MaxEntriesPerApp: -1,
			Provider:         "aws",
		})
		if !IsInvalidConfig(err) {
			t.Fatalf("disabled=%t: expected invalid config error, got %#v", disabled, err)
		}
	}
}

func Test_EnsureCreated_zeroMaxEntriesKeepsOnlyLatest(t *testing.T) {
	// The latest is the highest stable version, not the newest pre-release.
	const indexYAML = `entries:
  foo:
  - name: foo
    version: 1.0.0
    created: "2026-09-01T00:00:00Z"
  - name: foo
    version: 1.1.0
    created: "2026-09-02T00:00:00Z"
  - name: foo
    version: 1.2.0-rc.1
    created: "2026-09-03T00:00:00Z"
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(indexYAML))
	}))
	defer srv.Close()

	catalog := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "giantswarm",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1alpha1.CatalogSpec{
			Storage: v1alpha1.CatalogSpecStorage{URL: srv.URL, Type: "helm"},
		},
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.AppCatalogEntry{}, &v1alpha1.AppCatalogEntryList{})

	// An entry created earlier with a higher limit is deleted.
	ctrlClient := fake.NewClientBuilder().WithScheme(s).WithObjects(
		&v1alpha1.AppCatalogEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "giantswarm-foo-1.0.0",
				Namespace: metav1.NamespaceDefault,
				Labels: map[string]string{
					label.CatalogName: "giantswarm",
					label.ManagedBy:   key.AppCatalogEntryManagedBy(project.Name()),
				},
			},
		},
	).Build()

	r, err := New(Config{
		K8sClient: k8sclienttest.NewClients(k8sclienttest.ClientsConfig{CtrlClient: ctrlClient}),
		Logger:    microloggertest.New(),

		MaxEntriesPerApp: 0,
		Provider:         "aws",
		UniqueApp:        true,
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

	expected := []string{"giantswarm-foo-1.1.0"}
	if !reflect.DeepEqual(remaining, expected) {
		t.Fatalf("remaining entries = %v, want %v", remaining, expected)
	}
}

func Test_metadataURL(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expectedURL string
		expectedOK  bool
	}{
		{name: "nil annotations"},
		{name: "no metadata annotation", annotations: map[string]string{"foo": "bar"}},
		{
			name:        "current annotation",
			annotations: map[string]string{"io.giantswarm.application.metadata": "https://new"},
			expectedURL: "https://new",
			expectedOK:  true,
		},
		{
			name:        "legacy annotation",
			annotations: map[string]string{"application.giantswarm.io/metadata": "https://legacy"},
			expectedURL: "https://legacy",
			expectedOK:  true,
		},
		{
			name: "current annotation wins over legacy",
			annotations: map[string]string{
				"io.giantswarm.application.metadata": "https://new",
				"application.giantswarm.io/metadata": "https://legacy",
			},
			expectedURL: "https://new",
			expectedOK:  true,
		},
		{
			name: "empty current annotation falls back to legacy",
			annotations: map[string]string{
				"io.giantswarm.application.metadata": "",
				"application.giantswarm.io/metadata": "https://legacy",
			},
			expectedURL: "https://legacy",
			expectedOK:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url, ok := metadataURL(tc.annotations)
			if url != tc.expectedURL || ok != tc.expectedOK {
				t.Fatalf("metadataURL() = (%q, %t), want (%q, %t)", url, ok, tc.expectedURL, tc.expectedOK)
			}
		})
	}
}

func Test_getDesiredAppCatalogEntryCR_metadata(t *testing.T) {
	pythonTagged, err := os.ReadFile("testdata/main-python-tag.yaml")
	if err != nil {
		t.Fatal(err)
	}

	files := map[string][]byte{
		"/plain.yaml":         []byte("annotations:\n  application.giantswarm.io/team: honeybadger\nchartApiVersion: v2\nupstreamChartVersion: 1.2.3\n"),
		"/python-tagged.yaml": pythonTagged,
		"/invalid.yaml":       []byte("chartApiVersion: [v2\n"),
		"/invalid-key.yaml":   []byte("annotations:\n  application.giantswarm.io/restrictions/clusterSingleton: 'true'\n  application.giantswarm.io/team: honeybadger\nchartApiVersion: v2\n"),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/server-error.yaml" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("chartApiVersion: v2\n"))
			return
		}
		b, ok := files[req.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	tests := []struct {
		name                string
		annotations         map[string]string
		expectedAPIVersion  string
		expectedUpstream    string
		expectedAnnotations map[string]string
	}{
		{
			name:               "metadata from the current annotation, annotations merged",
			annotations:        map[string]string{"io.giantswarm.application.metadata": srv.URL + "/plain.yaml", "io.giantswarm.application.readme": "https://readme"},
			expectedAPIVersion: "v2",
			expectedUpstream:   "1.2.3",
			expectedAnnotations: map[string]string{
				"io.giantswarm.application.metadata": srv.URL + "/plain.yaml",
				"io.giantswarm.application.readme":   "https://readme",
				"application.giantswarm.io/team":     "honeybadger",
			},
		},
		{
			name:               "metadata from the legacy annotation",
			annotations:        map[string]string{"application.giantswarm.io/metadata": srv.URL + "/plain.yaml"},
			expectedAPIVersion: "v2",
			expectedUpstream:   "1.2.3",
			expectedAnnotations: map[string]string{
				"application.giantswarm.io/metadata": srv.URL + "/plain.yaml",
				"application.giantswarm.io/team":     "honeybadger",
			},
		},
		{
			name:               "main.yaml with a Python YAML tag keeps its string annotations",
			annotations:        map[string]string{"io.giantswarm.application.metadata": srv.URL + "/python-tagged.yaml", "artifacthub.io/links": "from index"},
			expectedAPIVersion: "v2",
			expectedAnnotations: map[string]string{
				"io.giantswarm.application.metadata": srv.URL + "/python-tagged.yaml",
				"artifacthub.io/links":               "from index",
				"artifacthub.io/license":             "Apache-2.0",
				"application.giantswarm.io/audience": "customers",
				"application.giantswarm.io/managed":  "true",
				"application.giantswarm.io/team":     "bumblebee",
			},
		},
		{
			name:                "unparseable main.yaml falls back to the index data",
			annotations:         map[string]string{"io.giantswarm.application.metadata": srv.URL + "/invalid.yaml"},
			expectedAPIVersion:  "v1",
			expectedAnnotations: map[string]string{"io.giantswarm.application.metadata": srv.URL + "/invalid.yaml"},
		},
		{
			name:               "annotation keys Kubernetes rejects are dropped",
			annotations:        map[string]string{"io.giantswarm.application.metadata": srv.URL + "/invalid-key.yaml"},
			expectedAPIVersion: "v2",
			expectedAnnotations: map[string]string{
				"io.giantswarm.application.metadata": srv.URL + "/invalid-key.yaml",
				"application.giantswarm.io/team":     "honeybadger",
			},
		},
		{
			name:                "server error falls back to the index data",
			annotations:         map[string]string{"io.giantswarm.application.metadata": srv.URL + "/server-error.yaml"},
			expectedAPIVersion:  "v1",
			expectedAnnotations: map[string]string{"io.giantswarm.application.metadata": srv.URL + "/server-error.yaml"},
		},
		{
			name:                "missing main.yaml falls back to the index data",
			annotations:         map[string]string{"io.giantswarm.application.metadata": srv.URL + "/missing.yaml"},
			expectedAPIVersion:  "v1",
			expectedAnnotations: map[string]string{"io.giantswarm.application.metadata": srv.URL + "/missing.yaml"},
		},
		{
			name:               "no metadata annotation",
			expectedAPIVersion: "v1",
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

			entryCR, err := r.getDesiredAppCatalogEntryCR(t.Context(), cr, e, true)
			if err != nil {
				t.Fatalf("unexpected error: %#v", err)
			}
			if entryCR.Spec.Chart.APIVersion != tc.expectedAPIVersion {
				t.Fatalf("chart API version = %q, want %q", entryCR.Spec.Chart.APIVersion, tc.expectedAPIVersion)
			}
			if entryCR.Spec.Chart.UpstreamChartVersion != tc.expectedUpstream {
				t.Fatalf("upstream chart version = %q, want %q", entryCR.Spec.Chart.UpstreamChartVersion, tc.expectedUpstream)
			}
			if diff := cmp.Diff(tc.expectedAnnotations, entryCR.Annotations); diff != "" {
				t.Fatalf("annotations (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_newAppCatalogEntries_fetchesEachMetadataOnce(t *testing.T) {
	var mu sync.Mutex
	requests := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		requests[req.URL.Path]++
		mu.Unlock()
		_, _ = w.Write([]byte("chartApiVersion: v2\n"))
	}))
	defer srv.Close()

	day := func(d int) metav1.Time {
		return metav1.NewTime(time.Date(2026, 9, d, 0, 0, 0, 0, time.UTC))
	}
	newEntry := func(version string, created metav1.Time) entry {
		return entry{
			Annotations: map[string]string{"io.giantswarm.application.metadata": srv.URL + "/" + version},
			Created:     created,
			Name:        "hello-world",
			Version:     version,
		}
	}

	i := index{Entries: map[string][]entry{
		"hello-world": {newEntry("1.0.0", day(1)), newEntry("1.1.0", day(2))},
	}}

	r := &Resource{logger: microloggertest.New(), maxEntriesPerApp: 5}
	cr := v1alpha1.Catalog{ObjectMeta: metav1.ObjectMeta{Name: "giantswarm", Namespace: "default"}}

	entryCRs, err := r.newAppCatalogEntries(t.Context(), cr, i)
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}
	if len(entryCRs) != 2 {
		t.Fatalf("got %d entries, want 2", len(entryCRs))
	}
	if diff := cmp.Diff(map[string]int{"/1.0.0": 1, "/1.1.0": 1}, requests); diff != "" {
		t.Fatalf("main.yaml requests (-want +got):\n%s", diff)
	}

	latest := entryCRs["giantswarm-hello-world-1.1.0"]
	if latest == nil || latest.Labels[pkglabel.Latest] != "true" {
		t.Fatalf("expected giantswarm-hello-world-1.1.0 to be labelled latest, got %#v", latest)
	}
	if other := entryCRs["giantswarm-hello-world-1.0.0"]; other == nil || other.Labels[pkglabel.Latest] != "false" {
		t.Fatalf("expected giantswarm-hello-world-1.0.0 not to be labelled latest, got %#v", other)
	}
}
