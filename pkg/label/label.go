// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package label

import (
	"fmt"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/app-operator/v7/pkg/project"
)

const (
	// Latest label is added to appcatalogentry CRs to filter for the most
	// recent release.
	Latest = "latest"
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

// ClusterSelector selects all apps with the specified cluster ID. Unless they
// use the special version `0.0.0` and are processed by the management cluster
// instance of app-operator.
func ClusterSelector(clusterID string) labels.Selector {
	s := fmt.Sprintf("%s=%s,%s!=%s", label.Cluster, clusterID, label.AppOperatorVersion, project.ManagementClusterAppVersion())

	selector, err := labels.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse selector %#q with error %#q", s, err))
	}

	return selector
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
