package appcatalog

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/microerror"
)

type index struct {
	APIVersion string             `yaml:"apiVersion"`
	Entries    map[string][]entry `yaml:"entries"`
	Generated  time.Time          `yaml:"generated"`
}

type entry struct {
	APIVersion  string    `yaml:"apiVersion"`
	AppVersion  string    `yaml:"appVersion"`
	Created     time.Time `yaml:"created"`
	Description string    `yaml:"description"`
	Digest      string    `yaml:"digest"`
	Name        string    `yaml:"name"`
	Urls        []string  `yaml:"urls"`
	Version     string    `yaml:"version"`
}

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

// GetLatestVersion returns the latest version in the specified catalog.
func GetLatestVersion(ctx context.Context, catalog, app string) (string, error) {
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

func getIndex(catalog string) (index, error) {
	indexURL := fmt.Sprintf("https://giantswarm.github.io/%s/index.yaml", catalog)
	resp, err := http.Get(indexURL)
	if err != nil {
		return index{}, microerror.Mask(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return index{}, microerror.Mask(err)
	}

	var i index
	err = yaml.Unmarshal(body, &i)
	if err != nil {
		return i, microerror.Mask(err)
	}

	return i, nil
}
