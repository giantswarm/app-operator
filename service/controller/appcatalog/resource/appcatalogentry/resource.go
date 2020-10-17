package appcatalogentry

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	pkglabel "github.com/giantswarm/app-operator/v2/pkg/label"
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
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("getting appcatalogentries for appcatalog %#q", cr.Name))

	currentEntryCRs := map[string]*v1alpha1.AppCatalogEntry{}

	lo := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.ManagedBy, project.Name(), pkglabel.CatalogName, cr.Name),
	}
	entries, err := r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(metav1.NamespaceDefault).List(ctx, lo)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for _, entry := range entries.Items {
		currentEntryCRs[entry.Name] = entry.DeepCopy()
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("got %d appcatalogentries for appcatalog %#q", len(currentEntryCRs), cr.Name))

	return currentEntryCRs, nil
}

func getIndex(storageURL string) (index, error) {
	indexURL := fmt.Sprintf("%s/index.yaml", storageURL)

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

	return i, nil
}

func equals(current, desired *v1alpha1.AppCatalogEntry) bool {
	if current.Name != desired.Name {
		return false
	}
	if !reflect.DeepEqual(current.Spec, desired.Spec) {
		return false
	}
	if !reflect.DeepEqual(current.Labels, desired.Labels) {
		return false
	}

	return true
}

func parseTime(created string) (*metav1.Time, error) {
	rawTime, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return nil, microerror.Maskf(executionFailedError, "wrong timestamp format %#q", created)
	}
	timeVal := metav1.NewTime(rawTime)

	return &timeVal, nil
}
