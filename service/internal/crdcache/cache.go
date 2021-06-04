package crdcache

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/app/v5/pkg/crd"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	gocache "github.com/patrickmn/go-cache"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	expiration = 4 * time.Hour
)

type Config struct {
	// Dependencies.
	Logger micrologger.Logger

	// Settings.
	githubToken string
}

type Resource struct {
	// Dependencies.
	cache     *gocache.Cache
	crdGetter *crd.CRDGetter
	logger    micrologger.Logger
}

// New creates a new configured clients resource.
func New(config Config) (*Resource, error) {
	var err error
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var crdGetter *crd.CRDGetter
	{
		c := crd.Config{
			Logger:      config.Logger,
			GitHubToken: config.githubToken,
		}

		crdGetter, err = crd.NewCRDGetter(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r := &Resource{
		// Dependencies.
		cache:     gocache.New(expiration, expiration/2),
		crdGetter: crdGetter,
		logger:    config.Logger,
	}

	return r, nil
}

func (r *Resource) LoadCRD(ctx context.Context, group, kind string) (*apiextensionsv1.CustomResourceDefinition, error) {
	k := fmt.Sprintf("%s/%s", group, kind)

	if v, ok := r.cache.Get(k); ok {
		c, ok := v.(*apiextensionsv1.CustomResourceDefinition)
		if !ok {
			return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &apiextensionsv1.CustomResourceDefinition{}, v)
		}

		return c, nil
	}

	crdResource, err := r.crdGetter.LoadCRD(ctx, group, kind)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.cache.SetDefault(k, crdResource)

	return crdResource, nil
}
