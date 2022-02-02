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
	cache      *gocache.Cache
	httpClient *http.Client
	logger     micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.HTTPClientTimeout == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.HTTPClientTimeout must not be empty", config)
	}

	// Set client timeout to prevent leakages.
	httpClient := &http.Client{
		Timeout: time.Second * time.Duration(config.HTTPClientTimeout),
	}

	r := &Resource{
		cache:      gocache.New(expiration, expiration/2),
		httpClient: httpClient,
		logger:     config.Logger,
	}

	return r, nil
}

func (r *Resource) GetIndex(ctx context.Context, storageURL string) (*Index, error) {
	indexURL := fmt.Sprintf("%s/index.yaml", strings.TrimRight(storageURL, "/"))

	if v, ok := r.cache.Get(indexURL); ok {
		i, ok := v.(Index)
		if !ok {
			return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", Index{}, v)
		}

		return &i, nil
	}

	r.logger.Debugf(ctx, "getting index %#q", indexURL)

	// We use https in catalog URLs so we can disable the linter in this case.
	resp, err := r.httpClient.Get(indexURL) // #nosec
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

	r.cache.SetDefault(indexURL, i)

	r.logger.Debugf(ctx, "got index %#q", indexURL)

	return &i, nil
}
