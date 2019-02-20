// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package label

const (
	// AppOperatorVersion is used to determine if the custom resource is
	// supported by this version of the operatorkit resource.
	AppOperatorVersion = "app-operator.giantswarm.io/version"

	// ChartOperatorVersion is set for chart CRs managed by the operator.
	ChartOperatorVersion = "chart-operator.giantswarm.io/version"

	// ManagedBy is set for Kubernetes resources managed by the operator.
	ManagedBy = "giantswarm.io/managed-by"
)

// ProcessLabels ensures the chart-operator.giantswarm.io/version label is
// present and the app-operator.giantswarm.io/version label is removed. It
// also ensures the giantswarm.io/managed-by label is accurate.
//
// Any other labels added to the app custom resource are passed on to the chart
// custom resource.
func ProcessLabels(projectName, chartCustomResourceVersion string, inputLabels map[string]string) map[string]string {
	// These labels are required.
	labels := map[string]string{
		ManagedBy: projectName,
	}

	if chartCustomResourceVersion != "" {
		labels[ChartOperatorVersion] = chartCustomResourceVersion
	}

	for k, v := range inputLabels {
		// These labels must be removed.
		if k != ManagedBy && k != AppOperatorVersion {
			labels[k] = v
		}
	}

	return labels
}
