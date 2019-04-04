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

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

// Config represents the configuration used to create a new appcatalog service.
type Config struct {
	// Dependencies.
	G8sClient versioned.Interface
	Logger    micrologger.Logger

	// Settings.
	WatchNamespace string
}

// AppCatalog implements the appcatalog service.
type AppCatalog struct {
	// Dependencies.
	g8sClient versioned.Interface
	logger    micrologger.Logger

	// Settings.
	watchNamespace string
}

// New creates a new configured appcatalog service.
func New(config Config) (*AppCatalog, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	a := &AppCatalog{
		g8sClient: config.G8sClient,
		logger:    config.Logger,

		watchNamespace: config.WatchNamespace,
	}

	return a, nil
}

// GetCatalogForApp gets the appCatalog CR specified in the provided app CR.
func (a *AppCatalog) GetCatalogForApp(ctx context.Context, customResource v1alpha1.App) (*v1alpha1.AppCatalog, error) {
	catalogName := key.CatalogName(customResource)

	a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for appCatalog %#q in namespace %#q", catalogName, a.watchNamespace))

	appCatalog, err := a.g8sClient.ApplicationV1alpha1().AppCatalogs("default").Get(catalogName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "appCatalog %#q in namespace %#q", catalogName, "default")
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found appCatalog %#q in namespace %#q", catalogName, a.watchNamespace))

	return appCatalog, nil
}
