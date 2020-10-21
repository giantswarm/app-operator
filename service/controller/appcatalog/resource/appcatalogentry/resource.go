package appcatalogentry

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkglabel "github.com/giantswarm/app-operator/v2/pkg/label"
	"github.com/giantswarm/app-operator/v2/pkg/project"
)

const (
	Name = "appcatalogentry"
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
