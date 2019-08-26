package tarball

import (
	"fmt"
	"net/url"
	"path"

	"github.com/giantswarm/microerror"
)

func GenerateTarballURL(baseURL string, appName string, version string) (string, error) {
	if baseURL == "" || appName == "" || version == "" {
		return "", microerror.Maskf(executionFailedError, "baseURL %#q, appName %#q, release %#q should not be empty", baseURL, appName, version)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", microerror.Mask(err)
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("%s-%s.tgz", appName, version))
	return u.String(), nil
}
