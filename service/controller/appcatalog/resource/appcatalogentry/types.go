package appcatalogentry

import "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"

type index struct {
	Entries   map[string][]entry `json:"entries"`
	Generated string             `json:"generated"`
}

type entry struct {
	AppVersion string   `json:"appVersion"`
	Created    string   `json:"created"`
	Home       string   `json:"home"`
	Icon       string   `json:"icon"`
	Name       string   `json:"name"`
	Urls       []string `json:"urls"`
	Version    string   `json:"version"`
}

type metadata struct {
	Restrictions v1alpha1.AppCatalogEntrySpecRestrictions `json:"restrictions"`
}
