// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package label

const (
	// AppOperatorVersion is used to determine if the custom resource is
	// supported by this version of the operatorkit resource.
	AppOperatorVersion = "app-operator.giantswarm.io/version"

	// ChartOperatorVersion is set for chart CRs managed by the operator.
	ChartOperatorVersion = "chart-operator.giantswarm.io/version"

	// Cluster label is a new style label for ClusterID
	Cluster = "giantswarm.io/cluster"

	// Organization label denotes guest cluster's organization ID as displayed
	// in the front-end.
	Organization = "giantswarm.io/organization"

	// ManagedBy is set for Kubernetes resources managed by the operator.
	ManagedBy = "giantswarm.io/managed-by"
)
