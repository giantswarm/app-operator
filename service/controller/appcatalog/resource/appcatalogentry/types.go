package appcatalogentry

import (
	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type index struct {
	Entries   map[string][]entry `json:"entries"`
	Generated string             `json:"generated"`
}

type entry struct {
	Annotations map[string]string `json:"annotations"`
	AppVersion  string            `json:"appVersion"`
	Created     metav1.Time       `json:"created"`
	Home        string            `json:"home"`
	Icon        string            `json:"icon"`
	Name        string            `json:"name"`
	Urls        []string          `json:"urls"`
	Version     string            `json:"version"`
	SemVer      semver.Version
}

type appMetadata struct {
	Annotations     map[string]string                         `json:"annotations"`
	ChartAPIVersion string                                    `json:"chartApiVersion"`
	DataCreated     *metav1.Time                              `json:"dataCreated"`
	Restrictions    *v1alpha1.AppCatalogEntrySpecRestrictions `json:"restrictions"`
}
