package appcatalogentry

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/app-operator/v5/pkg/project"
)

const (
	Name = "appcatalogentry"

	apiVersion           = "application.giantswarm.io/v1alpha1"
	communityCatalogType = "community"
	kindCatalog          = "Catalog"
	kindAppCatalogEntry  = "AppCatalogEntry"
	maxEntriesPerApp     = 5
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	MaxEntriesPerApp int
	UniqueApp        bool
}

type Resource struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	maxEntriesPerApp int
	uniqueApp        bool
}

// New creates a new configured tcnamespace resource.
func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.MaxEntriesPerApp == 0 {
		config.MaxEntriesPerApp = maxEntriesPerApp
	}

	r := &Resource{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		maxEntriesPerApp: config.MaxEntriesPerApp,
		uniqueApp:        config.UniqueApp,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

func (r *Resource) getCurrentEntryCRs(ctx context.Context, cr v1alpha1.Catalog) (map[string]*v1alpha1.AppCatalogEntry, error) {
	r.logger.Debugf(ctx, "getting current appcatalogentries for catalog %#q", cr.Name)

	currentEntryCRs := map[string]*v1alpha1.AppCatalogEntry{}

	entryLabels, err := labels.Parse(fmt.Sprintf("%s=%s,%s=%s", label.ManagedBy, key.AppCatalogEntryManagedBy(project.Name()), label.CatalogName, cr.Name))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	entries := &v1alpha1.AppCatalogEntryList{}
	err = r.k8sClient.CtrlClient().List(ctx, entries, &client.ListOptions{LabelSelector: entryLabels})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for _, entry := range entries.Items {
		currentEntryCRs[entry.Name] = entry.DeepCopy()
	}

	r.logger.Debugf(ctx, "got %d current appcatalogentries for catalog %#q", len(currentEntryCRs), cr.Name)

	return currentEntryCRs, nil
}

func (r *Resource) getIndex(ctx context.Context, storageURL string) (index, error) {
	indexURL := fmt.Sprintf("%s/index.yaml", strings.TrimRight(storageURL, "/"))

	r.logger.Debugf(ctx, "getting index.yaml from %#q", indexURL)

	// We use https in catalog URLs so we can disable the linter in this case.
	resp, err := http.Get(indexURL) // #nosec
	if err != nil {
		return index{}, microerror.Mask(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return index{}, microerror.Mask(err)
	}

	var i index

	err = yaml.Unmarshal(body, &i)
	if err != nil {
		return i, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "got index.yaml from %#q", indexURL)

	return i, nil
}

func (r *Resource) getMetadata(ctx context.Context, mainURL string) ([]byte, error) {
	eventName := "pull_metadata_file"

	t := prometheus.NewTimer(histogram.WithLabelValues(eventName))
	defer t.ObserveDuration()

	r.logger.Debugf(ctx, "getting main.yaml from %#q", mainURL)

	// We use https in catalog URLs so we can disable the linter in this case.
	resp, err := http.Get(mainURL) // #nosec
	if err != nil {
		return nil, microerror.Mask(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		r.logger.Debugf(ctx, "no main.yaml generated at %#q", mainURL)
		return nil, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "got main.yaml from %#q", mainURL)

	return body, nil
}

func parseMetadata(rawMetadata []byte) (*appMetadata, error) {
	var m appMetadata

	err := yaml.Unmarshal(rawMetadata, &m)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &m, nil
}

// copyAppCatalogEntry creates a new AppCatalogEntry object based on the current entry,
// so later we don't need to show unnecessary differences.
func copyAppCatalogEntry(current *v1alpha1.AppCatalogEntry) *v1alpha1.AppCatalogEntry {
	newChart := &v1alpha1.AppCatalogEntry{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       kindAppCatalogEntry,
		},
	}

	newChart.Name = current.Name
	newChart.Namespace = current.Namespace
	newChart.OwnerReferences = current.OwnerReferences

	newChart.Annotations = current.Annotations
	newChart.Labels = current.Labels
	newChart.Spec = *current.Spec.DeepCopy()

	return newChart
}
