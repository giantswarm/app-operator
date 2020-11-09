package appcatalog

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/app/v3/pkg/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "appcatalog"
)

// Config represents the configuration used to create a new appcatalog resource.
type Config struct {
	// Dependencies.
	G8sClient versioned.Interface
	Logger    micrologger.Logger
}

// Resource implements the appcatalog resource.
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

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for appCatalog %#q", catalogName))

	appCatalog, err := r.g8sClient.ApplicationV1alpha1().AppCatalogs().Get(ctx, catalogName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return microerror.Maskf(notFoundError, "appCatalog %#q", catalogName)
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found appCatalog %#q", catalogName))
	cc.AppCatalog = *appCatalog

	return nil
}
