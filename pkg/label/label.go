// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package label

const (
	// App label identifies the app being deployed.
	App = "app"

	// AppOperatorVersion is used to determine if the custom resource is
	// supported by this version of the operatorkit resource.
	AppOperatorVersion = "app-operator.giantswarm.io/version"

	// ChartOperatorVersion is set for chart CRs managed by the operator.
	ChartOperatorVersion = "chart-operator.giantswarm.io/version"

	// GiantSwarmPrefix is used to identify Giant Swarm specific labels.
	GiantSwarmPrefix = "giantswarm.io"

	// ManagedBy is set for Kubernetes resources managed by the operator.
	ManagedBy = "giantswarm.io/managed-by"
)
