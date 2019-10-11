package catalog

import (
	"context"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/microerror"
	"github.com/levigross/grequests"
)

// GetLatestChart returns the latest chart tarball file in the specified catalog.
func GetLatestChart(ctx context.Context, catalog, app string) (string, error) {
	index, err := getIndex(catalog)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var downloadURL string
	{
		entry, ok := index.Entries[app]
		if !ok {
			return "", microerror.Maskf(notFoundError, fmt.Sprintf("no app %q in index.yaml", app))
		}
		downloadURL = entry[0].Urls[0]
	}

	return downloadURL, nil
}

// GetLatestTag returns the latest tag in the specified catalog.
func GetLatestTag(ctx context.Context, catalog, app string) (string, error) {
	index, err := getIndex(catalog)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var version string
	{
		entry, ok := index.Entries[app]
		if !ok {
			return "", microerror.Maskf(notFoundError, fmt.Sprintf("no app %q in index.yaml", app))
		}
		version = entry[0].Version
	}

	return version, nil
}

func getIndex(catalog string) (*Index, error) {
	indexURL := fmt.Sprintf("https://giantswarm.github.io/%s/index.yaml", catalog)
	resp, err := grequests.Get(indexURL, nil)
	if err != nil {
		return &Index{}, microerror.Mask(err)
	}

	var index Index
	err = yaml.Unmarshal(resp.Bytes(), &index)
	if err != nil {
		return &Index{}, microerror.Mask(err)
	}

	return &index, nil
}
