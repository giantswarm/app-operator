// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package label

import (
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/app-operator/v3/pkg/project"
)

const (
	// Latest label is added to appcatalogentry CRs to filter for the most
	// recent release.
	Latest = "latest"

	// Watching is the label added to configmaps watched by the app value controller.
	Watching = "app-operator.giantswarm.io/watching"
)

func AppVersionSelector(unique bool) labels.Selector {
	var selector string

	if unique {
		selector = fmt.Sprintf("%s=%s", label.AppOperatorVersion, project.ManagementClusterAppVersion())
	} else {
		selector = fmt.Sprintf("%s!=%s", label.AppOperatorVersion, project.ManagementClusterAppVersion())
	}

	s, err := labels.Parse(selector)
	if err != nil {
		panic(fmt.Sprintf("failed to parse selector %#q with error %#q", s, err))
	}

	return s
}
