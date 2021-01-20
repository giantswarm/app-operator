// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package label

import (
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/app/v4/pkg/key"
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
	version := GetProjectVersion(unique)
	s := fmt.Sprintf("%s=%s", label.AppOperatorVersion, version)

	selector, err := labels.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse selector %#q with error %#q", s, err))
	}

	return selector
}

func ChartOperatorAppSelector(unique bool) string {
	var template string

	if unique {
		template = "%s=%s,%s=%s"
	} else {
		template = "%s!=%s,%s=%s"
	}

	return fmt.Sprintf(template, label.AppOperatorVersion,
		project.ManagementClusterAppVersion(),
		label.AppKubernetesName,
		key.ChartOperatorAppName)
}

func GetProjectVersion(unique bool) string {
	if unique {
		// When app-operator is deployed as a unique app it only processes
		// management cluster app CRs. These CRs always have the version label
		// app-operator.giantswarm.io/version: 0.0.0
		return project.ManagementClusterAppVersion()
	} else {
		return project.Version()
	}
}
