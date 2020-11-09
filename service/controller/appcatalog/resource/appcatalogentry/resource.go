package appcatalogentry

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/app/v3/pkg/key"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/app-operator/v2/pkg/project"
)

const (
	Name = "appcatalogentry"

	apiVersion           = "application.giantswarm.io/v1alpha1"
	communityCatalogType = "community"
	kindAppCatalog       = "AppCatalog"
	kindAppCatalogEntry  = "AppCatalogEntry"
	publicVisibilityType = "public"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	UniqueApp bool
}

type Resource struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger

	uniqueApp bool
}

// New creates a new configured tcnamespace resource.
func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		uniqueApp: config.UniqueApp,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

func (r *Resource) getCurrentEntryCRs(ctx context.Context, cr v1alpha1.AppCatalog) (map[string]*v1alpha1.AppCatalogEntry, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("getting current appcatalogentries for appcatalog %#q", cr.Name))

	currentEntryCRs := map[string]*v1alpha1.AppCatalogEntry{}

	lo := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.ManagedBy, key.AppCatalogEntryManagedBy(project.Name()), label.CatalogName, cr.Name),
	}
	entries, err := r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(metav1.NamespaceDefault).List(ctx, lo)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for _, entry := range entries.Items {
		currentEntryCRs[entry.Name] = entry.DeepCopy()
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("got %d current appcatalogentries for appcatalog %#q", len(currentEntryCRs), cr.Name))

	return currentEntryCRs, nil
}

func (r *Resource) getIndex(ctx context.Context, storageURL string) (index, error) {
	indexURL := fmt.Sprintf("%s/index.yaml", strings.TrimRight(storageURL, "/"))

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("getting index.yaml from %#q", indexURL))

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

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("got index.yaml from %#q", indexURL))

	return i, nil
}
