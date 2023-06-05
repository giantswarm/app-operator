package appcatalogentry

import (
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
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
	Description string            `json:"description"`
	Home        string            `json:"home"`
	Icon        string            `json:"icon"`
	Keywords    []string          `json:"keywords"`
	Name        string            `json:"name"`
	Urls        []string          `json:"urls"`
	Version     string            `json:"version"`
}

type appMetadata struct {
	Annotations          map[string]string                         `json:"annotations"`
	ChartAPIVersion      string                                    `json:"chartApiVersion"`
	DataCreated          *metav1.Time                              `json:"dataCreated"`
	Restrictions         *v1alpha1.AppCatalogEntrySpecRestrictions `json:"restrictions"`
	UpstreamChartVersion string                                    `json:"upstreamChartVersion"`
}
