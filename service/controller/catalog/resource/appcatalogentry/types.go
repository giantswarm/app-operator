package appcatalogentry

import (
	"encoding/json"
	"strings"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
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
	Annotations          metadataAnnotations                       `json:"annotations"`
	ChartAPIVersion      string                                    `json:"chartApiVersion"`
	DataCreated          *metav1.Time                              `json:"dataCreated"`
	Restrictions         *v1alpha1.AppCatalogEntrySpecRestrictions `json:"restrictions"`
	UpstreamChartVersion string                                    `json:"upstreamChartVersion"`
}

// metadataAnnotations holds the string annotations of a main.yaml. Values that
// are not strings are dropped: app-build-suite writes some of them with a
// Python-specific YAML tag (!!python/object/new:...BlockLiteralStr), which
// decodes to a list. Keys Kubernetes does not accept as annotation keys, such
// as application.giantswarm.io/restrictions/clusterSingleton, are dropped too,
// or the API server would reject the AppCatalogEntry.
type metadataAnnotations map[string]string

func (a *metadataAnnotations) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}

	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	if raw == nil {
		*a = nil
		return nil
	}

	annotations := make(metadataAnnotations, len(raw))
	for k, v := range raw {
		if len(validation.IsQualifiedName(strings.ToLower(k))) > 0 {
			continue
		}
		if s, ok := v.(string); ok {
			annotations[k] = s
		}
	}

	*a = annotations

	return nil
}
