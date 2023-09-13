package key

import (
	"fmt"
	"net/url"
	"path"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/microerror"
)

// GetRepositoryConfiguration returns Catalog CR supported repository details
// which are type and URL.
func GetRepositoryConfiguration(r interface{}) (string, string, error) {
	switch cr := r.(type) {
	case v1alpha1.CatalogSpecStorage:
		return cr.Type, cr.URL, nil
	case v1alpha1.CatalogSpecRepository:
		return cr.Type, cr.URL, nil
	default:
		return "", "", microerror.Maskf(
			wrongTypeError,
			"expected '%T' or '%T', got '%T'",
			v1alpha1.CatalogSpecStorage{}, v1alpha1.CatalogSpecRepository{},
			r,
		)
	}
}

// GetHelmRepositoryName turns repository type and URL into a name that
// suppose to uniquely identify a HelmRepository CR.
func GetHelmRepositoryName(c, t, u string) (string, error) {
	url, err := url.Parse(u)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Here we could possibly sanitize both hostname and path,
	// by making sure we always get the right values. IPv6 is an example
	// where currently this function will fail to deliver a good result.
	// Same goes for a path with query parameters. Fortunately we could rely,
	// for simplicity, on the fact that IPv6 addresses are rarely used directly,
	// much like the query parameters, if specifying them for index.yaml would
	// even make sense.
	hostname := url.Hostname()
	path := path.Base(url.Path)

	return fmt.Sprintf("%s-%s-%s-%s", c, t, hostname, path), nil
}
