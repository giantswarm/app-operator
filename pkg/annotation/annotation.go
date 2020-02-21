// Package annotation contains common Kubernetes metadata. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package annotation

const (
	// ChartOperatorConfigMapVersion is added to chart CRs. It has the resource
	// version for the chart values configmap so an update event is generated
	// when this changes.
	ChartOperatorConfigMapVersion = "chart-operator.giantswarm.io/configmap-version"

	// ChartOperatorNotes is for informational messages for resources
	// generated for chart-operator.
	ChartOperatorNotes = "chart-operator.giantswarm.io/notes"

	// ChartOperatorSecretVersion is added to chart CRs. It has the resource
	// version for the chart values secret so an update event is generated
	// when this changes.
	ChartOperatorSecretVersion = "chart-operator.giantswarm.io/secret-version"

	// ChartOperatorPrefix is the prefix for annotations that control logic inside chart-operator.
	ChartOperatorPrefix = "chart-operator.giantswarm.io"

	// CordonReason is the name of the annotation that indicates
	// the reason of why app-operator should not apply any update to this app CR.
	CordonReason = "app-operator.giantswarm.io/cordon-reason"

	// CordonUntil is the name of the annotation that indicates
	// the expiration date for this cordon rule.
	CordonUntil = "app-operator.giantswarm.io/cordon-until"
)
