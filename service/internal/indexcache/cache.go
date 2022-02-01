package indexcache

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	gocache "github.com/patrickmn/go-cache"
	"sigs.k8s.io/yaml"
)

const (
	expiration = 30 * time.Second
)

type Config struct {
	Logger micrologger.Logger

	HTTPClientTimeout time.Duration
}

type Resource struct {
	// Dependencies.
	cache  *gocache.Cache
	logger micrologger.Logger

	// Settings.
	httpClientTimeout time.Duration
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.HTTPClientTimeout == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.HTTPClientTimeout must not be empty", config)
	}

	r := &Resource{
		cache:  gocache.New(expiration, expiration/2),
		logger: config.Logger,

		httpClientTimeout: config.HTTPClientTimeout,
	}

	return r, nil
}

func (r *Resource) GetIndex(ctx context.Context, storageURL string) (*Index, error) {
	k := fmt.Sprintf("%s/index.yaml", strings.TrimRight(storageURL, "/"))

	if v, ok := r.cache.Get(k); ok {
		i, ok := v.(Index)
		if !ok {
			return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", Index{}, v)
		}

		return &i, nil
	}

	// We use https in catalog URLs so we can disable the linter in this case.
	resp, err := http.Get(k) // #nosec
	if err != nil {
		return nil, microerror.Mask(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var i Index
	err = yaml.Unmarshal(body, &i)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.cache.SetDefault(k, i)

	return &i, nil
}
