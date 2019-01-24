// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package label

const (
	// ChartOperatorVersion is set for chart CRs managed by the operator.
	ChartOperatorVersion = "chart-operator.giantswarm.io/version"

	// ManagedBy is set for Kubernetes resources managed by the operator.
	ManagedBy = "giantswarm.io/managed-by"

	// Version is used to determine if the custom resource is supported by this
	// version of the operatorkit resource.
	Version = "app-operator.giantswarm.io/version"
)
