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

// AppVersionSelector returns the label selector for this instance of
// app-operator.
func AppVersionSelector(unique bool) labels.Selector {
	var selector string

	if unique {
		// Unique instance watches all namespaces for app CRs with the unique
		// app version (0.0.0).
		selector = fmt.Sprintf("%s=%s", label.AppOperatorVersion, project.ManagementClusterAppVersion())
	} else {
		// Other instances watch the namespace they are running in but exclude
		// unique app CRs.
		selector = fmt.Sprintf("%s!=%s", label.AppOperatorVersion, project.ManagementClusterAppVersion())
	}

	s, err := labels.Parse(selector)
	if err != nil {
		panic(fmt.Sprintf("failed to parse selector %#q with error %#q", s, err))
	}

	return s
}

// ChartOperatorAppSelector returns the label selector for this instance of
// app-operator.
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
