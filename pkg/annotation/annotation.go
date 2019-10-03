// Package annotation contains common Kubernetes metadata. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package annotation

const (
	// CordonReason is the name of the annotation that indicates
	// the reason of why app-operator should not apply any update to this app CR.
	CordonReason = "app-operator.giantswarm.io/cordon-reason"

	// CordonUntil is the name of the annotation that indicates
	// the expiration date for this cordon rule.
	CordonUntil = "app-operator.giantswarm.io/cordon-until"

	// ForceHelmUpgrade is the name of the annotation that controls whether force
	// is used when upgrading the Helm release.
	ForceHelmUpgrade = "chart-operator.giantswarm.io/force-helm-upgrade"
)
