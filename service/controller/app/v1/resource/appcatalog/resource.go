package appcatalog

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

const (
	// Name is the identifier of the resource.
	Name = "appcatalogv1"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	G8sClient versioned.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	// Dependencies.
	g8sClient versioned.Interface
	logger    micrologger.Logger
}

// New creates a new configured appcatalog resource.
func New(config Config) (*Resource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		g8sClient: config.G8sClient,
		logger:    config.Logger,
	}

	return r, nil
}

func (*Resource) Name() string {
	return Name
}

// getCatalogForApp gets the appCatalog CR specified in the provided app CR.
func (r *Resource) getCatalogForApp(ctx context.Context, customResource v1alpha1.App) error {

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	catalogName := key.CatalogName(customResource)

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for appCatalog %#q in namespace %#q", catalogName, "default"))

	appCatalog, err := r.g8sClient.ApplicationV1alpha1().AppCatalogs().Get(catalogName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return microerror.Maskf(notFoundError, "appCatalog %#q in namespace %#q", catalogName, "default")
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found appCatalog %#q", catalogName))
	cc.AppCatalog = *appCatalog

	return nil
}
