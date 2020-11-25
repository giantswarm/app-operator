// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package label

import (
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/operatorkit/v4/pkg/controller"

	"github.com/giantswarm/app-operator/v2/pkg/project"
)

const (
	// Latest label is added to appcatalogentry CRs to filter for the most
	// recent release.
	Latest = "latest"

	// Watching is the label added to configmaps watched by the app value controller.
	Watching = "app-operator.giantswarm.io/watching"
)

func AppVersionSelector(unique bool) controller.Selector {
	return controller.NewSelector(func(labels controller.Labels) bool {
		if !labels.Has(label.AppOperatorVersion) {
			return false
		}
		if labels.Get(label.AppOperatorVersion) == getProjectVersion(unique) {
			return true
		}

		return false
	})
}

func getProjectVersion(unique bool) string {
	if unique {
		// When app-operator is deployed as a unique app it only processes
		// control plane app CRs. These CRs always have the version label
		// app-operator.giantswarm.io/version: 0.0.0
		return project.AppControlPlaneVersion()
	} else {
		return project.Version()
	}
}
