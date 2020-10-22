// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package label

import (
	"fmt"

	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/app-operator/v2/pkg/project"
)

const (
	// AppKubernetesName label is used to identify Kubernetes resources.
	AppKubernetesName = "app.kubernetes.io/name"

	// AppKubernetesVersion label is used to identify the version of Kubernetes
	// resources.
	AppKubernetesVersion = "app.kubernetes.io/version"

	// CatalogName label is used to identify resources belonging to a Giant Swarm
	// app catalog.
	CatalogName = "application.giantswarm.io/catalog"

	// CatalogType label is used to identify the type of Giant Swarm app catalog
	// e.g. stable or test.
	CatalogType = "application.giantswarm.io/catalog-type"

	// CatalogVisibility label is used to determine how Giant Swarm app catalogs
	// are displayed in the UX. e.g. public or internal.
	CatalogVisibility = "application.giantswarm.io/catalog-visibility"

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

func GetProjectVersion(unique bool) string {
	if unique {
		// When app-operator is deployed as a unique app it only processes
		// control plane app CRs. These CRs always have the version label
		// app-operator.giantswarm.io/version: 0.0.0
		return project.AppControlPlaneVersion()
	} else {
		return project.Version()
	}
}
