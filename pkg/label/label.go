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

// ProcessLabels merge the requiredLabels into inputLabels, also remove all excludedLabels elements.
func ProcessLabels(inputLabels, requiredLabels, excludedLabels map[string]string) map[string]string {
	labels := map[string]string{}

	for k, v := range inputLabels {
		// These labels must be removed.
		if _, ok := excludedLabels[k]; !ok {
			labels[k] = v
		}
	}

	// These labels must be included.
	for k, v := range requiredLabels {
		labels[k] = v
	}

	return labels
}
